package server

import (
	"fmt"
	"time"

	"github.com/HotPotatoC/twitter-clone/internal/module/auth"
	"github.com/HotPotatoC/twitter-clone/internal/module/relationship"
	"github.com/HotPotatoC/twitter-clone/internal/module/tweet"
	"github.com/HotPotatoC/twitter-clone/internal/module/user"
	"github.com/HotPotatoC/twitter-clone/pkg/cache"
	"github.com/HotPotatoC/twitter-clone/pkg/config"
	"github.com/HotPotatoC/twitter-clone/pkg/database"
	"github.com/HotPotatoC/twitter-clone/pkg/webserver"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.uber.org/zap"
)

type Server struct {
	config    *Config
	db        database.Database
	cache     cache.Cache
	log       *zap.SugaredLogger
	webserver webserver.WebServer
}

// New creates a new instance of a fiber web server
func New(webserver webserver.WebServer, db database.Database, cache cache.Cache, log *zap.SugaredLogger, config *Config) *Server {
	config.init()
	return &Server{
		config:    config,
		db:        db,
		cache:     cache,
		log:       log,
		webserver: webserver,
	}
}

func (s *Server) Listen() {
	s.initMiddlewares()
	s.initRoutes()
	if !fiber.IsChild() {
		s.log.Infof("Starting up %s %s:%s", s.config.AppName, s.config.Version, s.config.BuildID)
	}
	if err := s.webserver.Listen(fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)); err != nil {
		s.log.Error(err)
	}
}

func (s *Server) ListenTLS(certFile string, keyFile string) {
	s.initMiddlewares()
	s.initRoutes()
	if !fiber.IsChild() {
		s.log.Infof("Starting up %s %s:%s", s.config.AppName, s.config.Version, s.config.BuildID)
	}
	if err := s.webserver.ListenTLS(fmt.Sprintf("%s:%s", s.config.Host, s.config.Port), certFile, keyFile); err != nil {
		s.log.Error(err)
	}
}

func (s *Server) initMiddlewares() {
	s.webserver.Engine().Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowCredentials: true,
		AllowHeaders:     "Content-Type, X-CSRF-Token",
	}))

	s.webserver.Engine().Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
	}))

	s.webserver.Engine().Use(logger.New(logger.Config{
		Format: "${green}${time}${reset} | ${status} | ${cyan}${latency}${reset}	-	${host} | ${yellow}${method}${reset} | ${path} ${queryParams}\n",
	}))
}

func (s *Server) initRoutes() {
	csrfMiddleware := csrf.New(csrf.Config{
		CookieSecure: config.GetString("APP_ENV", "development") == "production",
		Expiration:   24 * time.Hour,
		KeyGenerator: func() string {
			return gonanoid.Must()
		},
	})
	authGroup := s.webserver.Engine().Group("/auth")
	tweetsGroup := s.webserver.Engine().Group("/tweets", csrfMiddleware)
	usersGroup := s.webserver.Engine().Group("/users", csrfMiddleware)
	relationshipGroup := s.webserver.Engine().Group("/relationships", csrfMiddleware)
	auth.Routes(authGroup, s.db, s.cache)
	tweet.Routes(tweetsGroup, s.db, s.cache)
	user.Routes(usersGroup, s.db, s.cache)
	relationship.Routes(relationshipGroup, s.db, s.cache)
}
