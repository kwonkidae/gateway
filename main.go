package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	krakendcors "github.com/devopsfaith/krakend-cors"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/router"
	krakendgin "github.com/devopsfaith/krakend/router/gin"
	jwt_lib "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	cors "gopkg.in/gin-contrib/cors.v1"
)

func main() {
	port := flag.Int("p", 0, "Port of the service")
	logLevel := flag.String("l", "ERROR", "Logging level")
	debug := flag.Bool("d", false, "Enable the debug")
	configFile := flag.String("c", "/etc/krakend/configuration.json", "Path to the configuration filename")
	flag.Parse()

	parser := config.NewParser()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}
	serviceConfig.Debug = serviceConfig.Debug || *debug
	if *port != 0 {
		serviceConfig.Port = *port
	}

	go runJWTGeneratorHTTPService("/token", "lawtalkSecret", "lawtalkAdmin", time.Hour*10, 7002)
	// f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	// 	log.Fatalf("error opening file: %v", err)
	// }
	// defer f.Close()

	logger, err := logging.NewLogger(*logLevel, os.Stdout, "[KRAKEND]")
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}

	mws := []gin.HandlerFunc{
		makeCors(serviceConfig.ExtraConfig),
		func(c *gin.Context) {
			// if c.Request.URL.Path == "/lawtalk/api/qna/question/alltypecount" {
			// 	jwt.Auth("lawtalkSecret")(c)
			// }
		},
	}

	// routerFactory := krakendgin.DefaultFactory(proxy.DefaultFactory(logger), logger)

	routerFactory := krakendgin.NewFactory(krakendgin.Config{
		Engine:       gin.Default(),
		ProxyFactory: customProxyFactory{logger, proxy.DefaultFactory(logger)},
		Middlewares:  mws,
		Logger:       logger,
		HandlerFactory: func(configuration *config.EndpointConfig, proxy proxy.Proxy) gin.HandlerFunc {
			return krakendgin.CustomErrorEndpointHandler(configuration, proxy, router.DefaultToHTTPError)
		},
	})

	routerFactory.New().Run(serviceConfig)

}

// customProxyFactory adds a logging middleware wrapping the internal factory
type customProxyFactory struct {
	logger  logging.Logger
	factory proxy.Factory
}

// New implements the Factory interface
func (cf customProxyFactory) New(cfg *config.EndpointConfig) (p proxy.Proxy, err error) {
	p, err = cf.factory.New(cfg)
	if err == nil {
		p = proxy.NewLoggingMiddleware(cf.logger, cfg.Endpoint)(p)
	}
	return
}

func makeCors(e config.ExtraConfig) gin.HandlerFunc {
	tmp := krakendcors.ConfigGetter(e)
	if tmp == nil {
		return nil
	}
	cfg, ok := tmp.(krakendcors.Config)
	if !ok {
		return nil
	}
	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		ExposeHeaders:    cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	})
}

func runJWTGeneratorHTTPService(resource, jwtSecret, jwtIssuer string, jwsTTL time.Duration, jwtPort int) {
	engine := gin.Default()
	engine.GET(fmt.Sprintf("%s/:id", resource), func(c *gin.Context) {
		token := jwt_lib.New(jwt_lib.GetSigningMethod("HS256"))
		token.Claims = jwt_lib.MapClaims{
			"Id":  c.Param("id"),
			"iss": jwtIssuer,
			"exp": time.Now().Add(jwsTTL).Unix(),
		}
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			c.JSON(500, gin.H{"message": "Could not generate token"})
		}
		c.JSON(200, gin.H{"token": tokenString})
	})
	log.Fatal(engine.Run(fmt.Sprintf(":%d", jwtPort)))
}
