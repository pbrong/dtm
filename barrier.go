package dtm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yedf/dtm/common"
)

type BusiFunc func(db *sql.DB) (interface{}, error)

type TransInfo struct {
	TransType  string
	Gid        string
	BranchID   string
	BranchType string
}

func (t *TransInfo) String() string {
	return fmt.Sprintf("transInfo: %s %s %s %s", t.TransType, t.Gid, t.BranchID, t.BranchType)
}

type BarrierModel struct {
	common.ModelBase
	TransInfo
}

func (BarrierModel) TableName() string { return "dtm_barrier.barrier" }

func insertBarrier(tx *sql.Tx, transType string, gid string, branchID string, branchType string) (int64, error) {
	if branchType == "" {
		return 0, nil
	}
	res, err := tx.Exec("insert into dtm_barrier.barrier(trans_type, gid, branch_id, branch_type) values(?,?,?,?)", transType, gid, branchID, branchType)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func ThroughBarrierCall(db *sql.DB, transType string, gid string, branchId string, branchType string, busiCall BusiFunc) (res interface{}, rerr error) {
	tx, rerr := db.BeginTx(context.Background(), &sql.TxOptions{})
	if rerr != nil {
		return
	}
	defer func() {
		if x := recover(); x != nil {
			tx.Rollback()
			panic(x)
		} else if rerr != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	originType := map[string]string{
		"cancel":     "action",
		"compensate": "action",
	}[branchType]
	originAffected, _ := insertBarrier(tx, transType, gid, branchId, originType)
	currentAffected, rerr := insertBarrier(tx, transType, gid, branchId, branchType)
	if currentAffected == 0 || (originType == "cancel" || originType == "compensate") && originAffected > 0 {
		return
	}
	res, rerr = busiCall(db)
	return
}
