package types

import (
	"github.com/jmoiron/sqlx/types"

	"github.com/crusttech/crust/internal/rules"
)

type (
	// Page - page structure
	Page struct {
		ID     uint64 `json:"pageID,string" db:"id"`
		SelfID uint64 `json:"selfID,string" db:"self_id"`

		NamespaceID uint64 `json:"namespaceID,string" db:"rel_namespace"`

		ModuleID uint64  `json:"moduleID,string" db:"module_id"`
		Module   *Module `json:"module,omitempty" db:"-"`

		Title       string `json:"title" db:"title"`
		Description string `json:"description" db:"description"`

		Blocks types.JSONText `json:"blocks" db:"blocks"`

		Children PageSet `json:"children,omitempty" db:"-"`

		Visible bool `json:"visible" db:"visible"`
		Weight  int  `json:"-" db:"weight"`
	}

	// Block - value of Page.Blocks ([]Block)
	Block struct {
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Options     types.JSONText `json:"options"`
		Kind        string         `json:"kind"`
		X           int            `json:"x"`
		Y           int            `json:"y"`
		Width       int            `json:"width"`
		Height      int            `json:"height"`
	}
)

// Resource returns a system resource ID for this type
func (p Page) PermissionResource() rules.Resource {
	return PagePermissionResource.AppendID(p.ID)
}
