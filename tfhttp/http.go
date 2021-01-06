package tfhttp

import (
	"context"
	"fmt"
	"github.com/brutella/hc/log"
	"github.com/gorilla/mux"
	tfaccessory "indievisible.org/toofar/accessory"
	"indievisible.org/toofar/config"
	"indievisible.org/toofar/platform"
	"indievisible.org/toofar/shelly"
	"net/http"
	"net/http/httputil"
	"time"
)

// Platform is the primary handle
type Platform struct {
	Running bool
}

// var r       *mux.Router
var srv http.Server

// Startup is called by the platform management to get things running
func (h Platform) Startup(c config.Config) platform.Control {
	// each platform should register its own routes
	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/shelly/{cmd}", shelly.Handler)

	// register some middleware to ensure that only local IP addresses can connect

	srv = http.Server{
		Addr:         c.HTTPAddress,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	go func() {
		log.Info.Printf("starting up HTTP control channel on %s", c.HTTPAddress)
		if err := srv.ListenAndServe(); err != nil {
			log.Info.Print(err)
		}
	}()

	return h
}

// Shutdown is called by the platform management to shut things down
func (h Platform) Shutdown() platform.Control {
	// log.Info.Print("shutting down HTTP control channel")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	srv.Shutdown(ctx)
	return h
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	log.Info.Print("HomeHandler requested")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	fmt.Fprint(w, "{ \"status\": \"OK\" }")
}

func debugMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		dump, _ := httputil.DumpRequest(req, false)
		log.Info.Print(string(dump))
		next.ServeHTTP(res, req)
	})
}

// AddAccessory - do not use, just satisfies the Platform interface
func (h Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	//
}

// GetAccessory - do not use, just satisfies the Platform interface
func (h Platform) GetAccessory(name string) (*tfaccessory.TFAccessory, bool) {
	return nil, false
}

// Background - just satisfies the Platform interface
func (h Platform) Background() {
	// nothing to do
}
