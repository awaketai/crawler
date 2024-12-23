package service

import (
	"context"

	pb "github.com/awaketai/crawler/goout/hello"
)

type Greet struct {
}

func (g *Greet) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Greeting = "Hello " + req.GetName()
	
	return nil
}
