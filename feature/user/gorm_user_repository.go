package user

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	gormhelpers "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type gormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) domain.UserRepository {
	return &gormUserRepository{db: db}
}

// allowedFilterColumns whitelists user columns clients may filter on.
// Excludes wallet_address, encrypted_wallet_private_key, and meta
// to prevent leaking secrets, bypassing soft delete, or running expensive
// JSONB predicates from untrusted input.
// deleted_at is intentionally included to enable trashed-user pagination
// via filters/sorts on the deleted_at column.
var allowedFilterColumns = map[string]bool{
	"id":           true,
	"name":         true,
	"email":        true,
	"role":         true,
	"number":       true,
	"birth_date":   true,
	"gender":       true,
	"created_at":   true,
	"updated_at":   true,
	"deleted_at":   true,
}

// allowedSortColumns whitelists user columns clients may sort on.
// deleted_at is intentionally included to enable sorting trashed users
// by their deletion timestamp.
var allowedSortColumns = map[string]bool{
	"id":         true,
	"name":       true,
	"email":      true,
	"role":       true,
	"gender":     true,
	"created_at": true,
	"updated_at": true,
	"deleted_at": true,
}

// Get retrieves users with pagination, search, filters, and sorts (batch operation)
// Returns: ([]User, int, error) - paginated slice, total count matching criteria, error
func (r *gormUserRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	db := r.db.WithContext(ctx).Unscoped().Model(&model.User{})

	if query != nil {
		if query.HasSearch() {
			db = db.Where(
				"LOWER(name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?) OR LOWER(number) LIKE LOWER(?) OR LOWER(wallet_address) LIKE LOWER(?)",
				"%"+query.Search+"%", "%"+query.Search+"%", "%"+query.Search+"%", "%"+query.Search+"%",
			)
		}

		if query.HasFilters() {
			db = gormhelpers.ApplyFilters(db, query.Filters, allowedFilterColumns, "")
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db = gormhelpers.ApplySorts(db, query, allowedSortColumns, "updated_at DESC", nil, "id ASC")
	db = gormhelpers.ApplyPagination(db, query)

	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}

	return domainUsers, int(total), nil
}

// Find retrieves a single user by ID, including soft-deleted users.
// Returns: (*User, error) - single entity lookup
func (r *gormUserRepository) Find(ctx context.Context, id string) (*domain.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Unscoped().First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	domainUser := user.ToDomain()
	return &domainUser, nil
}

// FindByEmails retrieves users by multiple emails, including soft-deleted users (batch operation)
// Returns: ([]User, error) - empty slice if no matches
func (r *gormUserRepository) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) {
	if len(emails) == 0 {
		return []domain.User{}, nil
	}
	var users []model.User
	if err := r.db.WithContext(ctx).Unscoped().Where("email IN ?", emails).Find(&users).Error; err != nil {
		return nil, err
	}
	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}
	return domainUsers, nil
}

// FindByRole retrieves all users with the specified role, including soft-deleted users
// Returns: ([]User, error) - empty slice if no matches
func (r *gormUserRepository) FindByRole(ctx context.Context, role domain.Role) ([]domain.User, error) {
	var users []model.User
	if err := r.db.WithContext(ctx).Unscoped().Where("role = ?", role).Find(&users).Error; err != nil {
		return nil, err
	}
	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}
	return domainUsers, nil
}

// FindByIds retrieves multiple users by IDs, including soft-deleted users (batch operation)
// Returns: ([]User, error) - batch lookup, empty slice if no matches
func (r *gormUserRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	if len(ids) == 0 {
		return []domain.User{}, nil
	}

	var users []model.User
	if err := r.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}

	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}

	return domainUsers, nil
}

// Update updates one or more users (partial update via non-zero fields)
// Returns: ([]User, error) - updated users, error
func (r *gormUserRepository) Update(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	if len(users) == 0 {
		return []domain.User{}, nil
	}
	if err := r.updateBatchCase(ctx, users); err != nil {
		return nil, err
	}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}
	return r.FindByIds(ctx, ids...)
}

// updateBatchCase builds and executes a single UPDATE statement using CASE expressions
// for each column that at least one user provides. Users that don't provide a column
// fall through to ELSE column (preserving the existing value). Eliminates the N+1
// per-user UPDATE pattern. Users sorted by Id for deterministic arg ordering.
func (r *gormUserRepository) updateBatchCase(ctx context.Context, users []domain.User) error {
	sort.Slice(users, func(i, j int) bool { return users[i].Id < users[j].Id })

	var clauses []string
	var allArgs [][]interface{}

	addCol := func(col string, getValue func(domain.User) (interface{}, bool)) {
		var pairs []interface{}
		for _, u := range users {
			if v, ok := getValue(u); ok {
				pairs = append(pairs, u.Id, v)
			}
		}
		if clause, args := gormhelpers.BuildCaseColumnSQL("id", col, pairs); clause != "" {
			clauses = append(clauses, clause)
			allArgs = append(allArgs, args)
		}
	}

	addCol("name", func(u domain.User) (interface{}, bool) {
		if u.Name != nil {
			return *u.Name, true
		}
		return nil, false
	})
	addCol("number", func(u domain.User) (interface{}, bool) {
		if u.Number != nil {
			return *u.Number, true
		}
		return nil, false
	})
	addCol("email", func(u domain.User) (interface{}, bool) {
		if u.Email != "" {
			return u.Email, true
		}
		return nil, false
	})
	addCol("birth_date", func(u domain.User) (interface{}, bool) {
		if u.BirthDate != nil {
			return *u.BirthDate, true
		}
		return nil, false
	})
	addCol("gender", func(u domain.User) (interface{}, bool) {
		if u.Gender != nil {
			return string(*u.Gender), true
		}
		return nil, false
	})
	addCol("meta", func(u domain.User) (interface{}, bool) {
		if u.Meta == nil {
			return nil, false
		}
		b, err := json.Marshal(u.Meta)
		if err != nil {
			return nil, false
		}
		return string(b), true
	})
	addCol("role", func(u domain.User) (interface{}, bool) {
		if u.Role != "" {
			return string(u.Role), true
		}
		return nil, false
	})
	addCol("wallet_address", func(u domain.User) (interface{}, bool) {
		if u.WalletAddress != "" {
			return u.WalletAddress, true
		}
		return nil, false
	})
	addCol("encrypted_wallet_private_key", func(u domain.User) (interface{}, bool) {
		if u.EncryptedWalletPrivateKey != "" {
			return u.EncryptedWalletPrivateKey, true
		}
		return nil, false
	})

	if len(clauses) == 0 {
		return nil
	}

	ids := make([]interface{}, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}
	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("users", "id", clauses, allArgs, ids, "updated_at = CURRENT_TIMESTAMP")
	return r.db.WithContext(ctx).Exec(sql, finalArgs...).Error
}

// UpdateRole batch updates roles for multiple users using efficient CASE statement
// Returns: ([]User, int64, error) - updated users slice, rows affected count, error
func (r *gormUserRepository) UpdateRole(ctx context.Context, users ...domain.User) ([]domain.User, int64, error) {
	if len(users) == 0 {
		return []domain.User{}, 0, nil
	}

	var pairs []interface{}
	for _, user := range users {
		pairs = append(pairs, user.Id, user.Role)
	}
	clause, clauseArgs := gormhelpers.BuildCaseColumnSQL("id", "role", pairs)

	ids := make([]interface{}, len(users))
	for i, user := range users {
		ids[i] = user.Id
	}

	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("users", "id", []string{clause}, [][]interface{}{clauseArgs}, ids, "updated_at = CURRENT_TIMESTAMP")

	result := r.db.WithContext(ctx).Exec(sql, finalArgs...)
	if err := result.Error; err != nil {
		return nil, 0, err
	}

	userIDs := make([]string, len(users))
	for i, user := range users {
		userIDs[i] = user.Id
	}
	updatedUsers, err := r.FindByIds(ctx, userIDs...)
	if err != nil {
		return nil, 0, err
	}

	return updatedUsers, result.RowsAffected, nil
}

// Store persists multiple users (batch operation)
// Returns: ([]User, error) - created users with generated IDs, error
func (r *gormUserRepository) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	// Generate ULID for users without ID
	for i := range users {
		if users[i].Id == "" {
			users[i].Id = ulid.Make().String()
		}
	}

	// Convert to model and batch INSERT
	modelUsers := make([]model.User, len(users))
	for i, u := range users {
		modelUsers[i] = model.FromDomainUser(u)
	}

	if err := r.db.WithContext(ctx).Create(&modelUsers).Error; err != nil {
		// Translate Postgres unique violation (23505) to domain code.
		// Catches concurrent batch creates with same email that raced past
		// the pre-check in userService.storeValidateEmails.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			emails := make([]string, len(users))
			for i, u := range users {
				emails[i] = u.Email
			}
			return nil, domain.NewError(
				domain.CodeUserStoreEmailDuplicateInDatabase,
				domain.WithMetadata("emails", emails),
				domain.WithError(err),
			)
		}
		return nil, err
	}

	// Convert back to domain and return
	created := make([]domain.User, len(modelUsers))
	for i, m := range modelUsers {
		created[i] = m.ToDomain()
	}
	return created, nil
}

// Delete deletes multiple users by IDs (batch operation).
// Pure soft-delete: GORM stamps `deleted_at`; the `role` column is preserved
// (last role before delete). On-chain role is set to RoleNone separately by
// userService.Delete via AuthorityService.UpdateUserRole. AuthMiddleware
// rejects trashed users at 401, so the preserved DB role cannot be exploited.
// Returns: (int64, error) - rows affected count, error
func (r *gormUserRepository) Delete(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).Delete(&model.User{}, "id IN ?", ids)
	return result.RowsAffected, result.Error
}

// Restore unsets deleted_at for trashed users (batch operation).
// Returns: (int64, error) - rows affected count, error
func (r *gormUserRepository) Restore(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).
		Unscoped().
		Model(&model.User{}).
		Where("id IN ?", ids).
		Update("deleted_at", nil)
	return result.RowsAffected, result.Error
}
