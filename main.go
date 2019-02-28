package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stefreak/terraform-state-store/stores"
)

var (
	store           stores.TerraformStateStore
	authUserKey     string = "user"
	authPasswordKey string = "password"
)

func getHelp(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"help": "bla",
	})
}

func getState(c *gin.Context) {
	state, error := store.RetrieveState(
		c.MustGet(authUserKey).(string),
		c.MustGet(authPasswordKey).(string),
		c.Param("id"))

	if error == stores.ErrorNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error": error.Error(),
		})
		return
	}
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": error.Error(),
		})
		return
	}

	c.String(http.StatusOK, "%s", state.Contents)
}

func setState(c *gin.Context) {
	data, error := c.GetRawData()

	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": error.Error(),
		})
		return
	}

	lockID, _ := c.GetQuery("ID")

	error = store.UpdateState(
		c.MustGet(authUserKey).(string),
		c.MustGet(authPasswordKey).(string),
		c.Param("id"),
		string(data),
		lockID,
	)

	if error == stores.ErrorLockedConflict {
		c.JSON(http.StatusConflict, gin.H{
			"error": error.Error(),
		})
		return
	}

	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated": true,
	})
}

type LockBody struct {
	ID string `json:"ID"`
}

func lockState(c *gin.Context) {
	var lock LockBody
	error := c.BindJSON(&lock)

	if error != nil {
		// gin already sent bad request response
		return
	}

	existingLockID, error := store.LockState(
		c.MustGet(authUserKey).(string),
		c.MustGet(authPasswordKey).(string),
		c.Param("id"),
		lock.ID,
	)

	if error == stores.ErrorLockedConflict {
		c.JSON(http.StatusConflict, gin.H{
			"error": error.Error(),
			"ID":    existingLockID,
		})
		return
	}

	if error == stores.ErrorNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error": error.Error(),
		})
		return
	}

	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"locked": true,
	})
}

func unlockState(c *gin.Context) {
	var lock LockBody

	// XXX: I believe it is a terraform bug, that lock ID is not always sent as body payload
	// when that is fixed we can change this line back to BindJSON
	c.ShouldBindJSON(&lock)

	error := store.UnlockState(
		c.MustGet(authUserKey).(string),
		c.MustGet(authPasswordKey).(string),
		c.Param("id"),
		lock.ID)

	if error == stores.ErrorLockedConflict {
		c.JSON(http.StatusConflict, gin.H{
			"error": error.Error(),
		})
		return
	}

	if error == stores.ErrorNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error": error.Error(),
		})
		return
	}

	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unlocked": true,
	})
}

func deleteState(c *gin.Context) {
	error := store.DeleteState(
		c.MustGet(authUserKey).(string),
		c.MustGet(authPasswordKey).(string),
		c.Param("id"),
	)

	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deleted": true,
	})
}

// BasicAuth ...
func BasicAuth() gin.HandlerFunc {
	realm := "Basic realm=" + strconv.Quote("Authorization Required")
	return func(c *gin.Context) {
		found := false
		var username string
		var password string

		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)
		if len(auth) == 2 {
			auth, error := base64.StdEncoding.DecodeString(auth[1])
			if error == nil {
				authParts := strings.SplitN(string(auth), ":", 2)
				if len(authParts) == 2 {
					username = authParts[0]
					password = authParts[1]
					found = true
				}
			}
		}

		if !found || store.ValidateAuth(username, password) != nil {
			// Credentials doesn't match, we return 401 and abort handlers chain.
			c.Header("WWW-Authenticate", realm)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Can be later retrieved with
		// c.MustGet(authUserKey).
		c.Set(authUserKey, username)
		c.Set(authPasswordKey, username)
	}
}

func main() {
	var (
		port = flag.Int("listen-port", 8080, "HTTP Server listen port")
	)

	store = stores.NewInMemoryTerraformStateStore()

	r := gin.Default()
	r.GET("/", getHelp)

	authorized := r.Group("/", BasicAuth())
	authorized.GET("/:id", getState)
	authorized.POST("/:id", setState)
	authorized.DELETE("/:id", deleteState)
	authorized.Handle("LOCK", "/:id", lockState)
	authorized.Handle("UNLOCK", "/:id", unlockState)

	r.Run(fmt.Sprintf(":%d", *port))
}
