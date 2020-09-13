package ziggurat

import (
	"context"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
)

type HttpServer struct {
	server  *http.Server
	retrier MessageRetrier
}

func (s *HttpServer) Start(ctx context.Context, config Config, retrier MessageRetrier, handlerMap TopicEntityHandlerMap) {
	router := httprouter.New()
	router.POST("/v1/dead_set/:topic_entity/:count", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		count, _ := strconv.Atoi(params.ByName("count"))
		retrier.Replay(config, handlerMap, params.ByName("topic_entity"), count)
		_, err := fmt.Fprintf(writer, "successfully replayed %d messages", count)
		if err != nil {
			writer.WriteHeader(500)
		}
	})

	server := &http.Server{Addr: ":8080", Handler: router}
	go func(server *http.Server) {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal().Err(err)
		}
	}(server)

	s.server = server

}

func (s *HttpServer) Stop() error {
	return s.server.Shutdown(context.Background())
}