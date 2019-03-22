package restapi

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/stefreak/terraform-state-store/auth"
	"github.com/stefreak/terraform-state-store/storage"
)

var (
	store        storage.StateStore
	validator    auth.Validator
	namespaceKey = "namespace"
)

// Run starts the web server
func Run(listen string, s storage.StateStore, v auth.Validator) {
	store = s
	validator = v

	r := gin.Default()
	r.GET("/", getHelp)

	authorized := r.Group("/v1/state", basicAuth())
	authorized.GET("/:id", getState)
	authorized.POST("/:id", setState)
	authorized.DELETE("/:id", deleteState)
	authorized.Handle("LOCK", "/:id", lockState)
	authorized.Handle("UNLOCK", "/:id", unlockState)

	r.Run(listen)
}

func getHelp(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"help": "bla",
	})
}

func getState(c *gin.Context) {
	state, err := store.Get(
		c.MustGet(namespaceKey).(string),
		c.Param("id"))

	if errors.Cause(err) == storage.ErrorNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.String(http.StatusOK, "%s", state.Contents)
}

func setState(c *gin.Context) {
	data, err := c.GetRawData()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	lockID, _ := c.GetQuery("ID")

	err = store.Update(
		c.MustGet(namespaceKey).(string),
		c.Param("id"),
		string(data),
		lockID,
	)

	if errors.Cause(err) == storage.ErrorLockedConflict {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated": true,
	})
}

type lockBody struct {
	ID string `json:"ID"`
}

func lockState(c *gin.Context) {
	var lock lockBody
	err := c.BindJSON(&lock)

	if err != nil {
		// gin already sent bad request response
		return
	}

	lockRetries := 120
	var existingLockID string
	for {
		existingLockID, err = store.Lock(
			c.MustGet(namespaceKey).(string),
			c.Param("id"),
			lock.ID)

		if errors.Cause(err) == storage.ErrorLockedConflict {
			select {
			case closed := <-c.Writer.CloseNotify():
				if closed {
					c.AbortWithStatus(400)
					return
				}
			case <-time.After(1 * time.Second):
				if lockRetries > 0 && !c.IsAborted() {
					lockRetries--
					continue
				}
			}

			// No more retries
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
				"ID":    existingLockID,
			})
			return
		}

		break
	}

	if errors.Cause(err) == storage.ErrorNotFound {
		// Terraform assumes that state already exists
		// Create empty state
		err = store.Update(
			c.MustGet(namespaceKey).(string),
			c.Param("id"),
			"",
			"")
		if err == nil {
			_, err = store.Lock(
				c.MustGet(namespaceKey).(string),
				c.Param("id"),
				lock.ID)
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"locked": true,
	})
}

func unlockState(c *gin.Context) {
	var lock lockBody

	// XXX: I believe it is a terraform bug, that lock ID is not always sent as body payload
	// when that is fixed we can change this line back to BindJSON (which would abort with bad request)
	c.ShouldBindJSON(&lock)

	var err error
	if lock.ID != "" {
		err = store.Unlock(
			c.MustGet(namespaceKey).(string),
			c.Param("id"),
			lock.ID)
	} else {
		err = store.ForceUnlock(
			c.MustGet(namespaceKey).(string),
			c.Param("id"))
	}

	if errors.Cause(err) == storage.ErrorLockedConflict {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	if errors.Cause(err) == storage.ErrorNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unlocked": true,
	})
}

func deleteState(c *gin.Context) {
	err := store.Delete(
		c.MustGet(namespaceKey).(string),
		c.Param("id"))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deleted": true,
	})
}

func basicAuth() gin.HandlerFunc {
	realm := "Basic realm=" + strconv.Quote("Authorization Required")
	return func(c *gin.Context) {
		found := false
		var username string
		var password string

		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)
		if len(auth) == 2 {
			auth, err := base64.StdEncoding.DecodeString(auth[1])
			if err == nil {
				authParts := strings.SplitN(string(auth), ":", 2)
				if len(authParts) == 2 {
					username = authParts[0]
					password = authParts[1]
					found = true
				}
			}
		}

		if found {
			namespace, err := validator.Validate(username, password)

			if err == nil {
				// Can be later retrieved with
				// c.MustGet(namespaceKey).
				c.Set(namespaceKey, namespace)
				return
			}
		}

		// Credentials doesn't match, we return 401 and abort handlers chain.
		c.Header("WWW-Authenticate", realm)
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}
