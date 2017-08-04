package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/DataDog/dd-trace-go/tracer"
	goredis "github.com/DataDog/dd-trace-go/tracer/contrib/go-redis"
	"github.com/DataDog/dd-trace-go/tracer/contrib/gorilla/muxtrace"
	"github.com/DataDog/dd-trace-go/tracer/contrib/sqltraced"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

func main() {
	t := tracer.NewTracerTransport(tracer.NewTransport("datadog", ""))
	r := newRouter(t)
	mt := muxtrace.NewMuxTracer("api", t)
	mt.HandleFunc(r.Router, "/", r.handler)
	log.Fatal(http.ListenAndServe(":8080", r))
}

type Router struct {
	*mux.Router
	redis *goredis.TracedClient
	pg    *sql.DB
}

func newRouter(t *tracer.Tracer) *Router {
	r := mux.NewRouter()

	redis := goredis.NewTracedClient(&redis.Options{
		Addr: "redis:6379",
	}, t, "redis")

	pg, err := sqltraced.OpenTraced(&pq.Driver{}, "host=postgres user=postgres dbname=postgres sslmode=disable", "postgres", t)
	if err != nil {
		panic(err)
	}

	return &Router{r, redis, pg}
}

func (r *Router) handler(w http.ResponseWriter, req *http.Request) {
	var name, population string

	// Link this call to redis to the previous to the request
	r.redis.SetContext(req.Context())

	// Count the number of hits on this enpoint
	n := r.redis.Incr("counter").Val()

	// Get the city associated to this number of hits
	err := r.pg.QueryRowContext(req.Context(), "SELECT name, population FROM city WHERE id = $1", n%20+1).Scan(&name, &population)
	if err != nil {
		log.Print(err)
		return
	}

	// Return the name of the city and its population
	fmt.Fprintf(w, "(%v hits) - City: %v, %v inhabitants", n, name, population)
}
