package model

type TreeNode struct {
	Path       string
	IsDir      bool
	L0Abstract string
	Children   []*TreeNode
}

type TreeOptions struct {
	MaxDepth         int32
	IncludeAbstracts bool
}
