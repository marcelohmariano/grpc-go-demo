package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	notev1 "github.com/marcelohmariano/grpc-go-demo/internal/gen/go/note/v1"
	"github.com/marcelohmariano/grpc-go-demo/internal/note"
)

var (
	serviceAddr = flag.String("addr", "localhost:50051", "note server address")
	serviceOp   = flag.String("op", "list", "operation (list, get, create, update, delete)")

	noteID      = flag.Int64("id", 0, "note ID")
	noteTitle   = flag.String("title", "", "note title")
	noteContent = flag.String("content", "", "note content")
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}

func run() error {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background())
	defer stop()

	c, err := note.NewAPIClient(ctx, *serviceAddr)
	if err != nil {
		return err
	}

	res := call(ctx, c, *serviceOp)
	return printResponse(res)
}

func call(ctx context.Context, c *note.APIClient, op string) proto.Message {
	var (
		res proto.Message
		err error
	)

	switch op {
	default: // list
		res, err = c.ListNotes(ctx, &emptypb.Empty{})
	case "get":
		r := &notev1.GetNoteRequest{NoteId: *noteID}
		res, err = c.GetNote(ctx, r)
	case "create":
		n := &notev1.Note{Title: *noteTitle, Content: *noteContent}
		r := &notev1.CreateNoteRequest{Note: n}
		res, err = c.CreateNote(ctx, r)
	case "update":
		n := &notev1.Note{Id: *noteID}
		m := &fieldmaskpb.FieldMask{}

		if *noteTitle != "" {
			n.Title = *noteTitle
			_ = m.Append(n, "title")
		}

		if *noteContent != "" {
			n.Content = *noteContent
			_ = m.Append(n, "content")
		}

		r := &notev1.UpdateNoteRequest{Note: n, UpdateMask: m}
		res, err = c.UpdateNote(ctx, r)
	case "delete":
		r := &notev1.DeleteNoteRequest{NoteId: *noteID}
		res, err = c.DeleteNote(ctx, r)
	}

	if err != nil {
		return status.Convert(err).Proto()
	}

	return res
}

func printResponse(res proto.Message) error {
	m := protojson.MarshalOptions{Multiline: true}

	b, err := m.Marshal(res)
	if err != nil {
		return err
	}

	fmt.Println(string(b))
	return nil
}
