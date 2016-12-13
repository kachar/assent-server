package main

import (
	"assent"
	"encoding/json"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/ory-am/ladon"
	"log"
	"net/http"
	"time"
)

type AssentResponse struct {
	IsAllowed bool
}
type ErrorResponse struct {
	StatusCode    int
	StatusMessage string
	ErrorMessage  string
}

type contextKey string

const (
	ladonRequestKey contextKey = "json-request"
)

func main() {

	commonHandlers := alice.New(recoverHandler, parsingHandler, loggingHandler)

	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/", commonHandlers.ThenFunc(Index)).Methods("GET")
	router.Handle("/check", commonHandlers.ThenFunc(CheckAccess)).Methods("POST")
	router.Handle("/policy", commonHandlers.ThenFunc(PolicyIndex)).Methods("POST")
	router.HandleFunc("/todos/{todoId}", TodoShow)
	router.NotFoundHandler = notFoundHandler()
	log.Fatal(http.ListenAndServe(":8080", router))
}

func parsingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var request *ladon.Request
		err := decoder.Decode(&request)
		if err != nil {
			panic(&ErrorResponse{
				StatusCode:   http.StatusBadRequest,
				ErrorMessage: "Can't parse input json2222" + err.Error(),
			})
		}
		log.Println(request)
		context.Set(r, ladonRequestKey, request)
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		request := getLadonRequest(r)
		log.Printf("[%s] %q %v %q\n", r.Method, r.URL.String(), t2.Sub(t1), request)
	}

	return http.HandlerFunc(fn)
}

func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %+v", err)
				var response *ErrorResponse
				if parsed, ok := err.(*ErrorResponse); ok {
					response = parsed
				} else {
					if identifier, ok := err.(string); ok {
						response = &ErrorResponse{
							StatusCode:   http.StatusInternalServerError,
							ErrorMessage: identifier,
						}
					}
				}
				ErrorResponseHandler(w, r, response)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func ErrorResponseHandler(w http.ResponseWriter, r *http.Request, e *ErrorResponse) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(e.StatusCode)
	e.StatusMessage = http.StatusText(e.StatusCode)
	json.NewEncoder(w).Encode(e)
}

func notFoundHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ErrorResponseHandler(w, r, &ErrorResponse{
			StatusCode:   http.StatusNotFound,
			ErrorMessage: "Resource not found",
		})
	}
	return http.HandlerFunc(fn)
}

func errorHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Response.StatusCode == http.StatusNotFound {
			notFoundResponse := &ErrorResponse{
				StatusCode:   http.StatusNotFound,
				ErrorMessage: "Resource not found",
			}
			json.NewEncoder(w).Encode(notFoundResponse)
		}
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome!")

}

func PolicyIndex(w http.ResponseWriter, r *http.Request) {
	// policy := assent.FetchDefaultPolicy()
	request := assent.FetchRequest()
	// bytes, _ := json.Marshal(request)
	w.Header().Add("Content-Type", "application/json")
	// w.Write(bytes)
	json.NewEncoder(w).Encode(request)
}

func getLadonRequest(r *http.Request) *ladon.Request {
	request, ok := context.Get(r, ladonRequestKey).(*ladon.Request)
	if !ok {
		panic(&ErrorResponse{
			StatusCode:   http.StatusBadRequest,
			ErrorMessage: "Can't parse input json",
		})
	}
	return request
}

func CheckAccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		panic(&ErrorResponse{
			StatusCode:   http.StatusMethodNotAllowed,
			ErrorMessage: "Use POST requests",
		})
	}

	// decoder := json.NewDecoder(r.Body)
	// var request *ladon.Request
	// err := decoder.Decode(&request)
	// if err != nil {
	// 	panic(&ErrorResponse{
	// 		StatusCode:   http.StatusBadRequest,
	// 		ErrorMessage: "Can't parse input json",
	// 	})
	// }
	request := getLadonRequest(r)

	response := &AssentResponse{}
	warden := assent.NewWarden()
	if err := warden.IsAllowed(request); err == nil {
		log.Println("Allowed")
		response.IsAllowed = true
	} else {
		log.Println("Access denied")
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func TodoShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	todoId := vars["todoId"]
	fmt.Fprintln(w, "Todo show:", todoId)
}
