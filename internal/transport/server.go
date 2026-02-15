package transport

//go:generate mockgen -source=server.go -destination=../service/mocks/mock_service.go -package=mocks ServiceDelayedNotifierInterface

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
	CreateNotification(*models.Notification) (string, error)
	GetNotificationStatus(id string) (string, error)
	DeleteNotification(id string) error
	ProcessNotification(nf *models.Notification) error
	GetAllNotifications() ([]*models.Notification, error)
}

type Server struct {
	ctx     context.Context
	cfg     *config.Config
	Service ServiceDelayedNotifierInterface
}

func NewServer(ctx context.Context, cfg *config.Config, srv ServiceDelayedNotifierInterface) *Server {
	return &Server{ctx: ctx, cfg: cfg, Service: srv}
}

func (s *Server) Run() error {
	eng := ginext.New("release")
	eng.Use(ginext.Logger())

	eng.Static("/static", "./web/static")
	eng.GET("/", s.ServeUI())

	v1 := eng.Group("/api/v1")
	v1.POST("/notify", s.NotifyCreateHandler())
	v1.GET("/notify/:id", s.NotifyGetHandler())
	v1.DELETE("/notify/:id", s.NotifyDeleteHandler())
	v1.GET("/notifications", s.GetAllNotificationsHandler())

	return eng.Run(s.cfg.GetString("HOST") + ":" + s.cfg.GetString("PORT"))
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
		status, err := s.Service.GetNotificationStatus(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": status})
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
		err := s.Service.DeleteNotification(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": fmt.Sprintf("notify %s is deleted", id)})
	}
}

func (s *Server) GetAllNotificationsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}
		}()
		notifications, err := s.Service.GetAllNotifications()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, notifications)
	}
}

func (s *Server) ServeUI() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.File("./web/templates/index.html")
	}
}
