// Copyright 2021 E99p1ant. All rights reserved.
// Use of this source code is governed by an AGPL-style
// license that can be found in the LICENSE file.

package db

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var _ ChallengesStore = (*challenges)(nil)

// Challenges is the default instance of the ChallengesStore.
var Challenges ChallengesStore

// ChallengesStore is the persistent interface for challenges.
type ChallengesStore interface {
	// Create creates a new challenge and persists to database.
	// It returns the challenge ID when challenge created.
	Create(ctx context.Context, opts CreateChallengeOptions) (uint, error)
	// BatchCreate creates challenges in batch.
	// It returns the challenges after they are created.
	BatchCreate(ctx context.Context, opts []CreateChallengeOptions) ([]*Challenge, error)
	// Get returns all the challenges.
	Get(ctx context.Context) ([]*Challenge, error)
	// GetByID returns the challenge with given id.
	// It returns ErrChallengeNotExists when not found.
	GetByID(ctx context.Context, id uint) (*Challenge, error)
	// GetByIDs returns the challenges with given ids.
	// It ignores the not exists challenge.
	GetByIDs(ctx context.Context, ids ...uint) ([]*Challenge, error)
	// Update updates the challenge with given id.
	Update(ctx context.Context, id uint, opts UpdateChallengeOptions) error
	// DeleteByID deletes the challenge with given id.
	DeleteByID(ctx context.Context, id uint) error
	// DeleteAll deletes all the challenges.
	DeleteAll(ctx context.Context) error
}

// NewChallengesStore returns a ChallengesStore instance with the given database connection.
func NewChallengesStore(db *gorm.DB) ChallengesStore {
	return &challenges{DB: db}
}

// Challenge represents the AWD challenge.
type Challenge struct {
	gorm.Model

	Title            string
	BaseScore        float64
	AutoRenewFlag    bool
	RenewFlagCommand string
}

type challenges struct {
	*gorm.DB
}

type CreateChallengeOptions struct {
	Title            string
	BaseScore        float64
	AutoRenewFlag    bool
	RenewFlagCommand string
}

var ErrChallengeAlreadyExists = errors.New("challenge already exits")

func (db *challenges) Create(ctx context.Context, opts CreateChallengeOptions) (uint, error) {
	var challenge Challenge
	if err := db.WithContext(ctx).Model(&Challenge{}).Where("title = ?", opts.Title).First(&challenge).Error; err == nil {
		return 0, ErrChallengeAlreadyExists
	} else if err != gorm.ErrRecordNotFound {
		return 0, errors.Wrap(err, "get")
	}

	c := &Challenge{
		Title:            opts.Title,
		BaseScore:        opts.BaseScore,
		AutoRenewFlag:    opts.AutoRenewFlag,
		RenewFlagCommand: opts.RenewFlagCommand,
	}
	if err := db.WithContext(ctx).Create(c).Error; err != nil {
		return 0, err
	}

	return c.ID, nil
}

func (db *challenges) BatchCreate(ctx context.Context, opts []CreateChallengeOptions) ([]*Challenge, error) {
	tx := db.Begin()

	challenges := make([]*Challenge, 0, len(opts))
	for _, option := range opts {
		var challenge Challenge
		if err := tx.WithContext(ctx).Model(&Challenge{}).Where("title = ?", option.Title).First(&challenge).Error; err == nil {
			tx.Rollback()
			return nil, ErrChallengeAlreadyExists
		} else if err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return nil, errors.Wrap(err, "get")
		}

		c := &Challenge{
			Title:            option.Title,
			BaseScore:        option.BaseScore,
			AutoRenewFlag:    option.AutoRenewFlag,
			RenewFlagCommand: option.RenewFlagCommand,
		}
		if err := tx.WithContext(ctx).Create(c).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		challenges = append(challenges, c)
	}

	return challenges, tx.Commit().Error
}

func (db *challenges) Get(ctx context.Context) ([]*Challenge, error) {
	var challenges []*Challenge
	return challenges, db.DB.WithContext(ctx).Model(&Challenge{}).Order("id ASC").Find(&challenges).Error
}

var ErrChallengeNotExists = errors.New("challenge does not exist")

func (db *challenges) GetByID(ctx context.Context, id uint) (*Challenge, error) {
	var challenge Challenge
	if err := db.WithContext(ctx).Model(&Challenge{}).Where("id = ?", id).First(&challenge).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrChallengeNotExists
		}
		return nil, errors.Wrap(err, "get")
	}
	return &challenge, nil
}

func (db *challenges) GetByIDs(ctx context.Context, ids ...uint) ([]*Challenge, error) {
	var challenges []*Challenge
	for _, id := range ids {
		var challenge Challenge
		if err := db.WithContext(ctx).Model(&Challenge{}).Where("id = ?", id).First(&challenge).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return nil, errors.Wrap(err, "get")
		}

		challenges = append(challenges, &challenge)
	}

	return challenges, nil
}

type UpdateChallengeOptions struct {
	Title            string
	BaseScore        float64
	AutoRenewFlag    bool
	RenewFlagCommand string
}

func (db *challenges) Update(ctx context.Context, id uint, opts UpdateChallengeOptions) error {
	return db.WithContext(ctx).Model(&Challenge{}).Where("id = ?", id).
		Select("Title", "BaseScore", "AutoRenewFlag", "RenewFlagCommand").
		Updates(&Challenge{
			Title:            opts.Title,
			BaseScore:        opts.BaseScore,
			AutoRenewFlag:    opts.AutoRenewFlag,
			RenewFlagCommand: opts.RenewFlagCommand,
		}).Error
}

func (db *challenges) DeleteByID(ctx context.Context, id uint) error {
	return db.WithContext(ctx).Delete(&Challenge{}, "id = ?", id).Error
}

func (db *challenges) DeleteAll(ctx context.Context) error {
	return db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Challenge{}).Error
}
