package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	ht "github.com/DataDog/dd-trace-go/tracer/contrib/net/httptrace"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	r := newRouter()
	r.HandleFunc("/", r.handler)
	log.Fatal(http.ListenAndServe(":8080", ht.NewTraceHandler(r, "web-backend", nil)))
}

type Router struct {
	*mux.Router
	redis *redis.Client
	pg    *sql.DB
}

func newRouter() *Router {
	r := mux.NewRouter()

	redis := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	pg, err := sql.Open("postgres", "host=postgres user=postgres dbname=postgres sslmode=disable")
	if err != nil {
		panic(err)
	}

	return &Router{r, redis, pg}
}

func (r *Router) handler(w http.ResponseWriter, req *http.Request) {
	var name, population string

	// Count the number of hits on this enpoint
	n := r.redis.Incr("counter").Val()

	// Get the city associated to this number of hits
	err := r.pg.QueryRow("SELECT name, population FROM city WHERE id = $1", n%20+1).Scan(&name, &population)
	if err != nil {
		log.Print(err)
		return
	}

	// Return the name of the city and its population
	fmt.Fprintf(w, "(%v hits) - City: %v, %v inhabitants", n, name, population)
}
