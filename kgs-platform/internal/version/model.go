package version

import "time"

type ChangeSet struct {
	EntitiesAdded    int
	EntitiesModified int
	EntitiesDeleted  int
	EdgesAdded       int
	EdgesModified    int
	EdgesDeleted     int
	CommitMessage    string
}

type GraphVersion struct {
	ID               string    `gorm:"column:id;primaryKey;size:64"`
	Namespace        string    `gorm:"column:namespace;index:idx_graph_versions_ns_created,priority:1;size:255;not null"`
	ParentID         string    `gorm:"column:parent_id;index;size:64"`
	CommitMessage    string    `gorm:"column:commit_message;size:1024"`
	EntitiesAdded    int       `gorm:"column:entities_added"`
	EntitiesModified int       `gorm:"column:entities_modified"`
	EntitiesDeleted  int       `gorm:"column:entities_deleted"`
	EdgesAdded       int       `gorm:"column:edges_added"`
	EdgesModified    int       `gorm:"column:edges_modified"`
	EdgesDeleted     int       `gorm:"column:edges_deleted"`
	RollbackFromID   string    `gorm:"column:rollback_from_id;size:64"`
	CreatedAt        time.Time `gorm:"column:created_at;index:idx_graph_versions_ns_created,priority:2"`
}

func (GraphVersion) TableName() string {
	return "graph_versions"
}

type DiffResult struct {
	FromVersionID    string
	ToVersionID      string
	EntitiesAdded    int
	EntitiesModified int
	EntitiesDeleted  int
	EdgesAdded       int
	EdgesModified    int
	EdgesDeleted     int
}
