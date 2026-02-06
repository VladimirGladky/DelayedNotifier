package transport

import (
	"DelayedNotifier/internal/models"
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/ginext"
)

type ServiceDelayedNotifierInterface interface {
	CreateNotification(*models.Notification) (int, error)
	GetNotification(id string) (*models.Notification, error)
	NotifyDeleteHandler(id string) error
}

type Server struct {
	ctx     context.Context
	cfg     *config.Config
	Service ServiceDelayedNotifierInterface
}

func NewServer(ctx context.Context, cfg *config.Config) *Server {
	return &Server{ctx: ctx, cfg: cfg}
}

func (s *Server) Run() error {
	eng := ginext.New("release")
	eng.Use(ginext.Logger())
	v1 := eng.Group("/api/v1")

	v1.POST("/notify", s.NotifyCreateHandler())
	v1.GET("/notify/:id", s.NotifyGetHandler())
	v1.DELETE("/notify/:id", s.NotifyDeleteHandler())

	return eng.Run(s.cfg.GetString("host") + ":" + s.cfg.GetString("port"))
}

func (s *Server) NotifyCreateHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error1"})
				return
			}
		}()
		var Request *models.Notification
		if err := c.ShouldBindJSON(&Request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		id, err := s.Service.CreateNotification(Request)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusOK, gin.H{"id": id})
	}
}
func (s *Server) NotifyGetHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error1"})
				return
			}
		}()
		id := c.Param("id")
		notification, err := s.Service.GetNotification(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, models.Notification{Id: notification.Id, Message: notification.Message, Time: notification.Time})
	}
}

func (s *Server) NotifyDeleteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error1"})
				return
			}
		}()
		id := c.Param("id")
		err := s.Service.NotifyDeleteHandler(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": fmt.Sprintf("notify %s is deleted", id)})
	}
}
