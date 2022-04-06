package grpcapi

import (
	"context"
	"database/sql"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/Sugar-pack/orders-manager/internal/db"
	"github.com/Sugar-pack/orders-manager/pkg/pb"
)

type OrderService struct {
	pb.OrdersManagerServiceServer
	dbConn *sqlx.DB
}

func (s OrderService) InsertOrder(ctx context.Context, order *pb.Order) (*pb.OrderTnxResponse, error) {
	logger := logging.FromContext(ctx)
	logger.Info("ReceiveOrder")
	orderID := uuid.New()
	txID := uuid.New()
	parseUserID, err := uuid.Parse(order.UserId)
	if err != nil {
		logger.Error("Error parsing user id ", err)

		return nil, status.Error(codes.Internal, "error parsing user id")
	}

	dbOrder := &db.Order{
		ID:        orderID,
		UserID:    parseUserID,
		Label:     order.Label,
		CreatedAt: order.CreatedAt.AsTime(),
	}

	transaction, err := s.dbConn.BeginTx(ctx, nil)
	if err != nil {
		logger.Error(err)

		return nil, fmt.Errorf("prepare tx failed %w", err)
	}
	defer func(tx *sql.Tx) {
		errRollback := tx.Rollback()
		if errRollback != nil {
			logger.Error(errRollback)
		}
	}(transaction)

	err = db.InsertUser(ctx, transaction, dbOrder)
	if err != nil {
		logger.Error("init transaction failed ", err)

		return nil, status.Error(codes.Internal, "init tx failed")
	}
	err = db.PrepareTransaction(ctx, transaction, txID)
	if err != nil {
		logger.Error("prepare tx failed ", err)

		return nil, status.Error(codes.Internal, "prepare tx failed")
	}
	err = transaction.Commit()
	if err != nil {
		logger.Error("commit tx failed ", err)

		return nil, status.Error(codes.Internal, "commit tx failed")
	}

	return &pb.OrderTnxResponse{
		Id:  orderID.String(),
		Tnx: txID.String(),
	}, nil
}
