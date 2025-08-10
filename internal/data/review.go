package data

import (
	"context"
	"errors"
	"review-service/internal/biz"
	"review-service/internal/data/model"
	"review-service/internal/data/query"

	"github.com/go-kratos/kratos/v2/log"
)

type reviewReop struct {
	data *Data
	log  *log.Helper
}

// NewReviewRepo .
func NewReviewRepo(data *Data, logger log.Logger) biz.ReviewRepo {
	return &reviewReop{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *reviewReop) SaveReview(ctx context.Context, review *model.ReviewInfo) (*model.ReviewInfo, error) {
	err := r.data.query.WithContext(ctx).ReviewInfo.Save(review)
	return review, err
}

// GetReviewByOrderID 根据订单ID查询评价
func (r *reviewReop) GetReviewByOrderID(ctx context.Context, orderID int64) ([]*model.ReviewInfo, error) {
	return r.data.query.ReviewInfo.WithContext(ctx).Where(r.data.query.ReviewInfo.OrderID.Eq((orderID))).Find()
}

// GetReview 根据评价ID获取评价
func (r *reviewReop) GetReview(ctx context.Context, reviewID int64) (*model.ReviewInfo, error) {
	return r.data.query.ReviewInfo.WithContext(ctx).Where(r.data.query.ReviewInfo.ReviewID.Eq(reviewID)).First()
}

// SaveReply 保存评价回复
func (r *reviewReop) SaveReply(ctx context.Context, reply *model.ReviewReplyInfo) (*model.ReviewReplyInfo, error) {
	//1. 数据校验
	//1.1 数据合法性校验（已回复的评价不允许商家再次回复）
	// 先用评价ID查库，看下是否已回复
	review, err := r.data.query.ReviewInfo.WithContext(ctx).Where(r.data.query.ReviewInfo.ReviewID.Eq(reply.ReviewID)).First()
	if err != nil {
		return nil, err
	}
	if review.HasReply == 1 {
		return nil, errors.New("该评价已回复")
	}

	//1.2 水平越权校验（A商家只能回复自己的，不能回复B商家的）
	// 举例子：用户A删除订单，userID + orderID 当条件去查询订单然后删除
	if reply.ReviewID != review.ReviewID {
		return nil, errors.New("水平越权")
	}

	//2. 更新数据库中的数据（评价回复表和评价表要同时跟新，涉及到事务操作）
	// 事务操作
	err = r.data.query.Transaction(func(tx *query.Query) error {
		// 回复表插入一条数据
		if err := tx.ReviewReplyInfo.WithContext(ctx).Save(reply); err != nil {
			r.log.WithContext(ctx).Errorf("SaveReply create reply fail, err:%v", err)
		}

		// 评价表更新 hasReply 字段
		if _, err := tx.ReviewInfo.
			WithContext(ctx).
			Where(tx.ReviewInfo.ReviewID.Eq(reply.ReviewID)).
			Update(tx.ReviewInfo.HasReply, 1); err != nil {
			r.log.WithContext(ctx).Errorf("SaveReply update review fail, err:%v", err)
		}

		return nil
	})

	//3. 返回
	return reply, err
}
