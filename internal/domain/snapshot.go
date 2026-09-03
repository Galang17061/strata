package domain

type SystemSnapshot struct {
	SystemSnapshotId string   `db:"system_snapshot_id" json:"systemSnapshotId"`
	RbdSystemId      string   `db:"rbd_system_id" json:"rbdSystemId"`
	Label            string   `db:"label" json:"label"`
	Kind             string   `db:"kind" json:"kind"`
	Payload          string   `db:"payload" json:"-"`
	CreatedBy        *string  `db:"created_by" json:"createdBy"`
	CreatedAt        DateTime `db:"created_at" json:"createdAt"`
}

type SnapshotCreateRequest struct {
	Label *string `json:"label"`
}

type SnapshotRestoreRequest struct {
	SystemName *string `json:"systemName"`
}

type SnapshotRestoreResult struct {
	ProjectId   string `json:"projectId"`
	RbdSystemId string `json:"rbdSystemId"`
}
