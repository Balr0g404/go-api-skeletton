package repositories

import (
	"context"
	"errors"

	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// userSortCols maps allowed sort field names to their DB column names.
var userSortCols = map[string]string{
	"id":         "id",
	"created_at": "created_at",
	"email":      "email",
	"first_name": "first_name",
	"last_name":  "last_name",
	"role":       "role",
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user. Returns gorm.ErrDuplicatedKey if the email already
// exists (detected from the database unique-index constraint, not only the
// application-level pre-check).
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return gorm.ErrDuplicatedKey
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) List(ctx context.Context, page, pageSize int, opts filtering.Options) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	q := applyUserFilters(r.db.WithContext(ctx).Model(&models.User{}), opts.Filters)
	q.Count(&total)

	offset := (page - 1) * pageSize
	err := applyUserSort(q, opts).Offset(offset).Limit(pageSize).Find(&users).Error
	return users, total, err
}

// ListCursor returns up to limit users with id > afterID, ordered by id ASC.
// Pass afterID = 0 to start from the beginning. Filters are applied but sort is
// always id ASC to preserve cursor semantics.
func (r *UserRepository) ListCursor(ctx context.Context, afterID uint, limit int, opts filtering.Options) ([]models.User, error) {
	var users []models.User
	q := applyUserFilters(r.db.WithContext(ctx), opts.Filters)
	err := q.Where("id > ?", afterID).Order("id ASC").Limit(limit).Find(&users).Error
	return users, err
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) bool {
	var count int64
	r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email).Count(&count)
	return count > 0
}

// Delete soft-deletes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}

// applyUserFilters applies whitelisted WHERE conditions to the query.
// The "active" field is handled as a boolean; all other fields use string equality.
func applyUserFilters(db *gorm.DB, filters map[string]string) *gorm.DB {
	for field, value := range filters {
		switch field {
		case "active":
			db = db.Where("active = ?", value == "true")
		case "email", "role":
			db = db.Where(field+" = ?", value)
		}
	}
	return db
}

// applyUserSort applies ORDER BY using a whitelisted column map.
func applyUserSort(db *gorm.DB, opts filtering.Options) *gorm.DB {
	col, ok := userSortCols[opts.Sort]
	if !ok {
		col = "id"
	}
	dir := "ASC"
	if opts.Order == filtering.OrderDesc {
		dir = "DESC"
	}
	return db.Order(col + " " + dir)
}
