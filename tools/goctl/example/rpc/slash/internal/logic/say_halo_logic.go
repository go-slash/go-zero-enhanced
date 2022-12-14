package logic

import (
	"context"

	"github.com/zeromicro/go-zero/tools/goctl/example/rpc/slash/internal/svc"
	"github.com/zeromicro/go-zero/tools/goctl/example/rpc/slash/pb/slash"

	"github.com/zeromicro/go-zero/core/logx"
)

type SayHaloLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSayHaloLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SayHaloLogic {
	return &SayHaloLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SayHaloLogic) SayHalo(in *slash.SayHaloReq) (*slash.SayHaloResp, error) {
	// todo: add your logic here and delete this line

	return &slash.SayHaloResp{}, nil
}
