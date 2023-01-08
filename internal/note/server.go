package note

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	notev1 "github.com/marcelohmariano/grpc-go-demo/internal/gen/go/note/v1"
)

var (
	seqID               atomic.Int64
	ErrNoteNotFound     = status.Error(codes.NotFound, "note not found")
	ErrNoteDataRequired = status.Error(codes.InvalidArgument, "note data required")
)

type server struct {
	kind       any
	shutdownFn func()
}

func (s server) serve(l net.Listener) error {
	srv := s.kind.(interface {
		Serve(l net.Listener) error
	})
	return srv.Serve(l)
}

func (s server) shutdown() {
	s.shutdownFn()
}

type APIServer struct {
	notev1.UnimplementedNoteAPIServer

	mu       sync.Mutex
	notes    map[int64]*notev1.Note
	grpcAddr string
	httpAddr string
}

func NewAPIServer(grpcAddr string, httpAddr string) *APIServer {
	return &APIServer{
		notes:    make(map[int64]*notev1.Note),
		grpcAddr: grpcAddr,
		httpAddr: httpAddr,
	}
}

func (s *APIServer) Listen(ctx context.Context) error {
	errc := make(chan error, 2)

	go func() {
		errc <- s.serveGRPC(ctx)
	}()
	go func() {
		errc <- s.serveHTTP(ctx)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errc:
		return err
	}
}

func (s *APIServer) ListNotes(_ context.Context, empty *emptypb.Empty) (*notev1.ListNotesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	notes := make([]*notev1.Note, 0, len(s.notes))
	for _, note := range s.notes {
		notes = append(notes, note)
	}

	return &notev1.ListNotesResponse{Notes: notes}, nil
}

func (s *APIServer) GetNote(_ context.Context, r *notev1.GetNoteRequest) (*notev1.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[r.GetNoteId()]
	if !ok {
		return nil, ErrNoteNotFound
	}
	return note, nil
}

func (s *APIServer) CreateNote(_ context.Context, r *notev1.CreateNoteRequest) (*notev1.Note, error) {
	note := r.GetNote()
	if note == nil {
		return nil, ErrNoteDataRequired
	}

	if note.Title == "" {
		return nil, requiredFieldError("title")
	}

	if note.Content == "" {
		return nil, requiredFieldError("content")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	note.Id = seqID.Add(1)
	note.CreatedAt = timestamppb.Now()
	note.UpdatedAt = note.CreatedAt

	s.notes[note.Id] = note
	return note, nil
}

func (s *APIServer) UpdateNote(_ context.Context, r *notev1.UpdateNoteRequest) (*notev1.Note, error) {
	rnote := r.GetNote()
	if rnote == nil {
		return nil, ErrNoteDataRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[rnote.GetId()]
	if !ok {
		return nil, ErrNoteNotFound
	}

	patch(r.GetUpdateMask(), map[string]func(){
		"title":   func() { note.Title = rnote.Title },
		"content": func() { note.Content = rnote.Content },
	})

	note.UpdatedAt = timestamppb.Now()
	return note, nil
}

func (s *APIServer) DeleteNote(_ context.Context, r *notev1.DeleteNoteRequest) (*emptypb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[r.GetNoteId()]
	if !ok {
		return nil, ErrNoteNotFound
	}
	delete(s.notes, note.Id)
	return &emptypb.Empty{}, nil
}

func requiredFieldError(name string) error {
	return status.Errorf(codes.InvalidArgument, "required field: %s", name)
}

func (s *APIServer) serveGRPC(ctx context.Context) error {
	grpcServer := grpc.NewServer()
	notev1.RegisterNoteAPIServer(grpcServer, s)
	reflection.Register(grpcServer)

	srv := server{
		kind:       grpcServer,
		shutdownFn: func() { grpcServer.GracefulStop() },
	}

	return s.listen(ctx, srv, "grpc", s.grpcAddr)
}

func (s *APIServer) serveHTTP(ctx context.Context) error {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err := notev1.RegisterNoteAPIHandlerFromEndpoint(ctx, mux, s.grpcAddr, opts)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Handler: mux,
	}

	srv := server{
		kind:       httpServer,
		shutdownFn: func() { _ = httpServer.Shutdown(ctx) },
	}

	return s.listen(ctx, srv, "http", s.httpAddr)
}

func (s *APIServer) listen(ctx context.Context, server server, protocol string, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("%s kind listening on %s", protocol, addr)

	errc := make(chan error, 1)
	go func() {
		errc <- server.serve(lis)
	}()

	select {
	case <-ctx.Done():
		server.shutdown()
		return nil
	case err := <-errc:
		return err
	}
}

func patch(mask *fieldmaskpb.FieldMask, m map[string]func()) {
	if mask == nil {
		for _, f := range m {
			f()
		}
	}

	for _, path := range mask.Paths {
		f, ok := m[path]
		if !ok {
			continue
		}
		f()
	}
}
