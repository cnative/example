package state

import "time"

//Report represents a report object
type Report struct {
	ID        string    `db:"id" json:"id,omitempty"`                 // id of the report
	Name      string    `db:"name" json:"name,omitempty"`             // name of the report
	Labels    string    `db:"labels" json:"labels,omitempty"`         // labels is json string of name value pairs
	CreatedBy string    `db:"created_by" json:"created_by,omitempty"` // who created report
	UpdatedBy string    `db:"updated_by" json:"updated_by,omitempty"` // who last updated the report
	CreatedAt time.Time `db:"created_at" json:"created_at,omitempty"` // when was the report created
	UpdatedAt time.Time `db:"updated_at" json:"updated_at,omitempty"` // when was the report last updated
}
