package models

type Category struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"size:100;not null;unique" json:"name"`
	Slug string `gorm:"size:100;not null;unique" json:"slug"`
}
