package barlogic

import (
	"context"

	"github.com/zeromicro/go-zero/tools/goctl/example/rpc/slash/internal/svc"
	"github.com/zeromicro/go-zero/tools/goctl/example/rpc/slash/pb/slash"

	"github.com/zeromicro/go-zero/core/logx"
)

type SayHiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSayHiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SayHiLogic {
	return &SayHiLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// group: bar
func (l *SayHiLogic) SayHi(in *slash.HiReq) (*slash.HiResp, error) {
	// todo: add your logic here and delete this line

	return &slash.HiResp{}, nil
}
