package foologic

import (
	"context"

	"github.com/zeromicro/go-zero/tools/goctl/example/rpc/slash/internal/svc"
	"github.com/zeromicro/go-zero/tools/goctl/example/rpc/slash/pb/slash"

	"github.com/zeromicro/go-zero/core/logx"
)

type SayHelloLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSayHelloLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SayHelloLogic {
	return &SayHelloLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// group: foo
func (l *SayHelloLogic) SayHello(in *slash.HelloReq) (*slash.HelloResp, error) {
	// todo: add your logic here and delete this line

	return &slash.HelloResp{}, nil
}
