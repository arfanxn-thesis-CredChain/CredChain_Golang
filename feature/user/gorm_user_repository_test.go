package user

import (
	"context"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRepo(t *testing.T) domain.UserRepository {
	t.Helper()
	return NewGormUserRepository(db.OpenInMemorySQLite(t))
}

func TestGormUserRepository_Store_AutoGeneratesULID(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID(""), fixtures.WithEmail("auto@x.com"))
	stored, err := repo.Store(context.Background(), u)
	assert.NoError(t, err)
	assert.Len(t, stored, 1)
	assert.NotEmpty(t, stored[0].Id)
}

func TestGormUserRepository_Store_PreservesProvidedID(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("custom-id"), fixtures.WithEmail("custom@x.com"))
	stored, err := repo.Store(context.Background(), u)
	assert.NoError(t, err)
	assert.Equal(t, "custom-id", stored[0].Id)
}

func TestGormUserRepository_Store_Batch(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithEmail("a@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithEmail("b@x.com"))
	stored, err := repo.Store(context.Background(), u1, u2)
	assert.NoError(t, err)
	assert.Len(t, stored, 2)
}

func TestGormUserRepository_Find(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("findme"), fixtures.WithEmail("find@x.com"))
	_, _ = repo.Store(context.Background(), u)

	found, err := repo.Find(context.Background(), "findme")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "find@x.com", found.Email)
}

func TestGormUserRepository_Find_NotFound(t *testing.T) {
	repo := newRepo(t)
	_, err := repo.Find(context.Background(), "nope")
	assert.Error(t, err)
}

func TestGormUserRepository_FindByEmails_Empty(t *testing.T) {
	repo := newRepo(t)
	users, err := repo.FindByEmails(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, users)
}

func TestGormUserRepository_FindByEmails_NoMatches(t *testing.T) {
	repo := newRepo(t)
	users, err := repo.FindByEmails(context.Background(), "missing@x.com")
	assert.NoError(t, err)
	assert.Empty(t, users)
}

func TestGormUserRepository_FindByEmails_Multiple(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("a@x.com")),
		fixtures.NewDomainUser(fixtures.WithEmail("b@x.com")),
		fixtures.NewDomainUser(fixtures.WithEmail("c@x.com")),
	)
	users, err := repo.FindByEmails(context.Background(), "a@x.com", "c@x.com")
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestGormUserRepository_FindByRole(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("a@x.com"), fixtures.WithRole(domain.RoleAdmin)),
		fixtures.NewDomainUser(fixtures.WithEmail("b@x.com"), fixtures.WithRole(domain.RoleHolder)),
	)
	admins, err := repo.FindByRole(context.Background(), domain.RoleAdmin)
	assert.NoError(t, err)
	assert.Len(t, admins, 1)
	assert.Equal(t, "a@x.com", admins[0].Email)
}

func TestGormUserRepository_FindByIds_Empty(t *testing.T) {
	repo := newRepo(t)
	got, err := repo.FindByIds(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestGormUserRepository_FindByIds_Multiple(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithID("id1"), fixtures.WithEmail("a@x.com")),
		fixtures.NewDomainUser(fixtures.WithID("id2"), fixtures.WithEmail("b@x.com")),
	)
	got, err := repo.FindByIds(context.Background(), "id1", "id2", "missing")
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestGormUserRepository_Update(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("upd"), fixtures.WithEmail("old@x.com"))
	_, _ = repo.Store(context.Background(), u)

	u.Email = "new@x.com"
	updated, err := repo.Update(context.Background(), u)
	assert.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "new@x.com", updated[0].Email)
}

func TestGormUserRepository_Delete_Empty(t *testing.T) {
	repo := newRepo(t)
	n, err := repo.Delete(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, 0, n)
}

func TestGormUserRepository_Delete(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("d1"), fixtures.WithEmail("a@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("d2"), fixtures.WithEmail("b@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2)

	n, err := repo.Delete(context.Background(), "d1", "d2")
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n)

	got, findErr := repo.Find(context.Background(), "d1")
	assert.NoError(t, findErr, "Find is unscoped: trashed users must still be returned post-delete")
	assert.NotNil(t, got)
	assert.NotNil(t, got.DeletedAt, "trashed user must carry DeletedAt timestamp")
}

func TestGormUserRepository_UpdateRole_BatchCASE(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithEmail("a@x.com"), fixtures.WithRole(domain.RoleHolder))
	u2 := fixtures.NewDomainUser(fixtures.WithID("u2"), fixtures.WithEmail("b@x.com"), fixtures.WithRole(domain.RoleHolder))
	_, _ = repo.Store(context.Background(), u1, u2)

	u1.Role = domain.RoleIssuer
	u2.Role = domain.RoleAdmin
	updated, n, err := repo.UpdateRole(context.Background(), u1, u2)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n)
	assert.Len(t, updated, 2)
	roles := map[string]domain.Role{updated[0].Id: updated[0].Role, updated[1].Id: updated[1].Role}
	assert.Equal(t, domain.RoleIssuer, roles["u1"])
	assert.Equal(t, domain.RoleAdmin, roles["u2"])
}

func TestGormUserRepository_UpdateRole_Empty(t *testing.T) {
	repo := newRepo(t)
	got, n, err := repo.UpdateRole(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, got)
	assert.EqualValues(t, 0, n)
}

func TestGormUserRepository_Get_DefaultOrder(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("a@x.com")),
	)
	time.Sleep(10 * time.Millisecond)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("b@x.com")),
	)
	q := &domainQuery.Query{Page: 1, Limit: 10}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, users, 2)
	assert.Equal(t, "b@x.com", users[0].Email, "newer user should come first with updated_at DESC")
	assert.Equal(t, "a@x.com", users[1].Email)
}

func TestGormUserRepository_Get_SearchByName(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithName("Alice"), fixtures.WithEmail("a@x.com")),
		fixtures.NewDomainUser(fixtures.WithName("Bob"), fixtures.WithEmail("b@x.com")),
	)
	q := &domainQuery.Query{Page: 1, Limit: 10, Search: "ali"}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, users, 1)
}

func TestGormUserRepository_Get_SearchCaseInsensitive(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("alice@x.com")),
		fixtures.NewDomainUser(fixtures.WithEmail("bob@x.com")),
	)
	q := &domainQuery.Query{Page: 1, Limit: 10, Search: "ALICE"}
	users, _, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Len(t, users, 1, "search must be case-insensitive")
}

func TestGormUserRepository_Get_SearchByNumber(t *testing.T) {
	repo := newRepo(t)
	num := "2209123456"
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithNumber(num), fixtures.WithEmail("num@x.com")),
		fixtures.NewDomainUser(fixtures.WithEmail("other@x.com")),
	)
	q := &domainQuery.Query{Page: 1, Limit: 10, Search: "2209123456"}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, users, 1)
	assert.Equal(t, "num@x.com", users[0].Email)
}

func TestGormUserRepository_Get_SearchByWalletAddress(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithWalletAddress("0xDEADBEEF"), fixtures.WithEmail("wallet@x.com")),
		fixtures.NewDomainUser(fixtures.WithEmail("other@x.com")),
	)
	q := &domainQuery.Query{Page: 1, Limit: 10, Search: "0xdeadbeef"}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, users, 1)
	assert.Equal(t, "wallet@x.com", users[0].Email)
}

func TestGormUserRepository_Get_PageLimit(t *testing.T) {
	repo := newRepo(t)
	for i := 0; i < 5; i++ {
		_, _ = repo.Store(context.Background(), fixtures.NewDomainUser())
	}
	q := &domainQuery.Query{Page: 1, Limit: 2}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, users, 2)
}

func TestGormUserRepository_Get_SortByName(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithName("Charlie"), fixtures.WithEmail("c@x.com")),
		fixtures.NewDomainUser(fixtures.WithName("Alice"), fixtures.WithEmail("a@x.com")),
		fixtures.NewDomainUser(fixtures.WithName("Bob"), fixtures.WithEmail("b@x.com")),
	)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Sorts: []domainQuery.Sort{{Column: "name", Order: domainQuery.SortAsc}},
	}
	users, _, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Len(t, users, 3)
	assert.Equal(t, "Alice", *users[0].Name)
	assert.Equal(t, "Bob", *users[1].Name)
	assert.Equal(t, "Charlie", *users[2].Name)
}

func TestGormUserRepository_Delete_SoftDelete_FindReturnsTrashed(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("sd1"), fixtures.WithEmail("sd1@x.com"))
	_, _ = repo.Store(context.Background(), u)
	_, err := repo.Delete(context.Background(), "sd1")
	assert.NoError(t, err)
	got, findErr := repo.Find(context.Background(), "sd1")
	assert.NoError(t, findErr, "Find is unscoped: trashed users must be returned")
	assert.NotNil(t, got)
	assert.NotNil(t, got.DeletedAt, "trashed user must carry DeletedAt timestamp")
}

func TestGormUserRepository_Delete_SoftDelete_IncludesTrashedInGet(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("sd2"), fixtures.WithEmail("sd2@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("sd3"), fixtures.WithEmail("sd3@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2)
	_, _ = repo.Delete(context.Background(), "sd2")
	users, total, err := repo.Get(context.Background(), &domainQuery.Query{})
	assert.NoError(t, err)
	assert.Equal(t, 2, total, "Get is always unscoped: trashed users must be included by default")
	assert.Len(t, users, 2)
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}
	assert.Contains(t, ids, "sd2")
	assert.Contains(t, ids, "sd3")
}

func TestGormUserRepository_Delete_SoftDelete_FilterNullExcludesTrashed(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("sd2a"), fixtures.WithEmail("sd2a@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("sd3a"), fixtures.WithEmail("sd3a@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2)
	_, _ = repo.Delete(context.Background(), "sd2a")
	users, total, err := repo.Get(context.Background(), &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "deleted_at", Operator: domainQuery.OperatorNull},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, total, "deleted_at IS NULL filter must exclude trashed users")
	assert.Len(t, users, 1)
	assert.Equal(t, "sd3a", users[0].Id)
}

func TestGormUserRepository_Delete_SoftDelete_FindByIdsReturnsTrashed(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("sd4"), fixtures.WithEmail("sd4@x.com"))
	_, _ = repo.Store(context.Background(), u)
	_, _ = repo.Delete(context.Background(), "sd4")
	found, err := repo.FindByIds(context.Background(), "sd4")
	assert.NoError(t, err)
	assert.Len(t, found, 1, "FindByIds is unscoped: trashed users must be returned")
	assert.NotNil(t, found[0].DeletedAt)
}

func TestGormUserRepository_Delete_SoftDelete_FindByEmailsReturnsTrashed(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("sd5"), fixtures.WithEmail("sd5@x.com"))
	_, _ = repo.Store(context.Background(), u)
	_, _ = repo.Delete(context.Background(), "sd5")
	found, err := repo.FindByEmails(context.Background(), "sd5@x.com")
	assert.NoError(t, err)
	assert.Len(t, found, 1, "FindByEmails is unscoped: trashed users must be returned")
	assert.NotNil(t, found[0].DeletedAt)
}

func TestGormUserRepository_Delete_SoftDelete_FindByRoleReturnsTrashed(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("sd6"), fixtures.WithEmail("sd6@x.com"), fixtures.WithRole(domain.RoleHolder))
	_, _ = repo.Store(context.Background(), u)
	_, _ = repo.Delete(context.Background(), "sd6")
	found, err := repo.FindByRole(context.Background(), domain.RoleHolder)
	assert.NoError(t, err)
	assert.Len(t, found, 1, "FindByRole is unscoped: trashed users must be returned (role preserved on delete)")
	assert.NotNil(t, found[0].DeletedAt)
}

func TestGormUserRepository_Update_BirthDateSet_PersistsValue(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("bd1"), fixtures.WithEmail("bd1@x.com"))
	_, _ = repo.Store(context.Background(), u)
	bd := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := repo.Update(context.Background(), domain.User{Id: "bd1", BirthDate: &bd})
	assert.NoError(t, err)
	got, _ := repo.Find(context.Background(), "bd1")
	assert.NotNil(t, got.BirthDate)
	assert.Equal(t, 1990, got.BirthDate.Year())
}

func TestGormUserRepository_Update_BirthDateNil_PreservesExisting(t *testing.T) {
	repo := newRepo(t)
	bd := time.Date(2000, 5, 15, 0, 0, 0, 0, time.UTC)
	u := fixtures.NewDomainUser(fixtures.WithID("bd2"), fixtures.WithEmail("bd2@x.com"))
	u.BirthDate = &bd
	_, _ = repo.Store(context.Background(), u)
	name := "Renamed"
	_, err := repo.Update(context.Background(), domain.User{Id: "bd2", Name: &name})
	assert.NoError(t, err)
	got, _ := repo.Find(context.Background(), "bd2")
	assert.NotNil(t, got.BirthDate, "nil pointer in update payload must not clear existing date")
	assert.Equal(t, 2000, got.BirthDate.Year())
}

func TestGormUserRepository_Update_BirthDateExplicitClear_SetsNull(t *testing.T) {
	t.Skip("explicit-clear-to-null is intentionally unsupported with struct-based Updates; revisit if product requires it")
}

func TestGormUserRepository_Update_GenderSet_PersistsValue(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("g1"), fixtures.WithEmail("g1@x.com"))
	_, _ = repo.Store(context.Background(), u)
	gender := domain.GenderFemale
	_, err := repo.Update(context.Background(), domain.User{Id: "g1", Gender: &gender})
	assert.NoError(t, err)
	got, _ := repo.Find(context.Background(), "g1")
	assert.NotNil(t, got.Gender)
	assert.Equal(t, domain.GenderFemale, *got.Gender)
}

func TestGormUserRepository_Update_GenderNil_PreservesExisting(t *testing.T) {
	repo := newRepo(t)
	gender := domain.GenderMale
	u := fixtures.NewDomainUser(fixtures.WithID("g2"), fixtures.WithEmail("g2@x.com"))
	u.Gender = &gender
	_, _ = repo.Store(context.Background(), u)
	name := "Renamed"
	_, err := repo.Update(context.Background(), domain.User{Id: "g2", Name: &name})
	assert.NoError(t, err)
	got, _ := repo.Find(context.Background(), "g2")
	assert.NotNil(t, got.Gender, "nil pointer in update payload must not clear existing gender")
	assert.Equal(t, domain.GenderMale, *got.Gender)
}

func TestGormUserRepository_Update_BatchCASE_MixedColumns(t *testing.T) {
	repo := newRepo(t)

	u1 := fixtures.NewDomainUser(fixtures.WithID("bc1"), fixtures.WithEmail("bc1@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("bc2"), fixtures.WithEmail("bc2@x.com"))
	u3 := fixtures.NewDomainUser(fixtures.WithID("bc3"), fixtures.WithEmail("bc3@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2, u3)

	name1 := "Alice"
	num2 := "99999"
	num3 := "88888"

	updated, err := repo.Update(context.Background(),
		domain.User{Id: "bc1", Name: &name1},
		domain.User{Id: "bc2", Number: &num2},
		domain.User{Id: "bc3", Number: &num3},
	)
	assert.NoError(t, err)
	assert.Len(t, updated, 3)

	byID := make(map[string]domain.User)
	for _, u := range updated {
		byID[u.Id] = u
	}

	assert.Equal(t, "Alice", *byID["bc1"].Name)
	assert.Equal(t, "bc1@x.com", byID["bc1"].Email)

	assert.Equal(t, "99999", *byID["bc2"].Number)
	assert.Equal(t, "bc2@x.com", byID["bc2"].Email)

	assert.Equal(t, "88888", *byID["bc3"].Number)
	assert.Equal(t, "bc3@x.com", byID["bc3"].Email)

	assert.Nil(t, byID["bc1"].Number)
	assert.Nil(t, byID["bc2"].Name)
	assert.Nil(t, byID["bc3"].Name)
}

func seedUsersForFilter(t *testing.T) domain.UserRepository {
	t.Helper()
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithID("f1"), fixtures.WithName("Alice"), fixtures.WithEmail("alice@example.com"), fixtures.WithRole(domain.RoleAdmin)),
		fixtures.NewDomainUser(fixtures.WithID("f2"), fixtures.WithName("Bob"), fixtures.WithEmail("bob@example.com"), fixtures.WithRole(domain.RoleIssuer)),
		fixtures.NewDomainUser(fixtures.WithID("f3"), fixtures.WithName("Charlie"), fixtures.WithEmail("charlie@example.com"), fixtures.WithRole(domain.RoleHolder)),
		fixtures.NewDomainUser(fixtures.WithID("f4"), fixtures.WithName("Dave"), fixtures.WithEmail("dave@example.com"), fixtures.WithRole(domain.RoleHolder)),
	)
	return repo
}

func TestGormUserRepository_Get_FilterEqualByRole(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("role", domainQuery.OperatorEqual, "holder")},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, users, 2)
	for _, u := range users {
		assert.Equal(t, domain.RoleHolder, u.Role)
	}
}

func TestGormUserRepository_Get_FilterNotEqualByRole(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("role", domainQuery.OperatorNotEqual, "holder")},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	for _, u := range users {
		assert.NotEqual(t, domain.RoleHolder, u.Role)
	}
}

func TestGormUserRepository_Get_FilterLikeByName(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("name", domainQuery.OperatorLike, "ali")},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Alice", *users[0].Name)
}

func TestGormUserRepository_Get_FilterILikeIsCaseInsensitive(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("name", domainQuery.OperatorILike, "ALI")},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Alice", *users[0].Name)
}

func TestGormUserRepository_Get_FilterInByID(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("id", domainQuery.OperatorIn, "f1", "f3")},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	got := map[string]bool{}
	for _, u := range users {
		got[u.Id] = true
	}
	assert.True(t, got["f1"])
	assert.True(t, got["f3"])
}

func TestGormUserRepository_Get_FilterNotInByID(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("id", domainQuery.OperatorNotIn, "f1", "f3")},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	for _, u := range users {
		assert.NotEqual(t, "f1", u.Id)
		assert.NotEqual(t, "f3", u.Id)
	}
}

func TestGormUserRepository_Get_FilterMultipleAreAnded(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{
			domainQuery.NewFilter("role", domainQuery.OperatorEqual, "holder"),
			domainQuery.NewFilter("name", domainQuery.OperatorLike, "char"),
		},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Charlie", *users[0].Name)
}

func TestGormUserRepository_Get_FilterDisallowedColumnIsIgnored(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("wallet_address", domainQuery.OperatorEqual, "0xdead")},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 4, total, "filter on non-allowlisted column must be silently dropped")
	assert.Len(t, users, 4)
}

func TestGormUserRepository_Get_FilterBetweenCreatedAt(t *testing.T) {
	repo := newRepo(t)
	now := time.Now()
	u := fixtures.NewDomainUser(fixtures.WithID("recent"), fixtures.WithEmail("recent@x.com"))
	u.CreatedAt = now
	_, _ = repo.Store(context.Background(), u)
	yesterday := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	tomorrow := now.Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("created_at", domainQuery.OperatorBetween, yesterday, tomorrow)},
	}
	_, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestGormUserRepository_Get_FilterCombinedWithSearch(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Search:  "example.com",
		Filters: []domainQuery.Filter{domainQuery.NewFilter("role", domainQuery.OperatorEqual, "issuer")},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Bob", *users[0].Name)
}

func TestGormUserRepository_Get_SortByEmailDesc(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Sorts: []domainQuery.Sort{{Column: "email", Order: domainQuery.SortDesc}},
	}
	users, _, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Len(t, users, 4)
	assert.Equal(t, "dave@example.com", users[0].Email)
	assert.Equal(t, "alice@example.com", users[3].Email)
}

func TestGormUserRepository_Get_SortByRole(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Sorts: []domainQuery.Sort{{Column: "role", Order: domainQuery.SortAsc}},
	}
	users, _, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Len(t, users, 4)
}

func TestGormUserRepository_Get_SortDisallowedColumnIsIgnored(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Sorts: []domainQuery.Sort{{Column: "wallet_address", Order: domainQuery.SortAsc}},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err, "disallowed sort column must be silently dropped, not crash")
	assert.Equal(t, 4, total)
	assert.Len(t, users, 4)
}

func TestGormUserRepository_Get_FilterPaginationCountIsConsistent(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 1,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("role", domainQuery.OperatorEqual, "holder")},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 2, total, "total must reflect filtered count, not page slice")
	assert.Len(t, users, 1, "items respect page limit")
}

func TestGormUserRepository_Get_PaginationPage2(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 2, Limit: 2,
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 4, total, "total must reflect all matching rows")
	assert.Len(t, users, 2, "page 2 with limit 2 should return remaining 2 items")
}

func TestGormUserRepository_Get_PaginationPage3Empty(t *testing.T) {
	repo := seedUsersForFilter(t)
	q := &domainQuery.Query{
		Page: 3, Limit: 2,
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 4, total, "total is unaffected by out-of-range page")
	assert.Empty(t, users, "page beyond data range should return empty slice")
}

// seedUsersForTrashed seeds 2 live users + 1 trashed user (soft-deleted).
// Returns repo plus the IDs in the order they were inserted.
func seedUsersForTrashed(t *testing.T) domain.UserRepository {
	t.Helper()
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithID("live-a"), fixtures.WithName("LiveA"), fixtures.WithEmail("livea@x.com"), fixtures.WithRole(domain.RoleHolder)),
		fixtures.NewDomainUser(fixtures.WithID("live-b"), fixtures.WithName("LiveB"), fixtures.WithEmail("liveb@x.com"), fixtures.WithRole(domain.RoleHolder)),
		fixtures.NewDomainUser(fixtures.WithID("trashed-c"), fixtures.WithName("TrashedC"), fixtures.WithEmail("trashedc@x.com"), fixtures.WithRole(domain.RoleHolder)),
	)
	_, _ = repo.Delete(context.Background(), "trashed-c")
	return repo
}

func TestGormUserRepository_Get_FilterDeletedAtNotNull_ReturnsOnlyTrashed(t *testing.T) {
	repo := seedUsersForTrashed(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("deleted_at", domainQuery.OperatorNotNull)},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, users, 1)
	assert.Equal(t, "trashed-c", users[0].Id)
	assert.NotNil(t, users[0].DeletedAt)
}

func TestGormUserRepository_Get_FilterDeletedAtNull_ReturnsOnlyLive(t *testing.T) {
	repo := seedUsersForTrashed(t)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("deleted_at", domainQuery.OperatorNull)},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	for _, u := range users {
		assert.Nil(t, u.DeletedAt)
	}
}

func TestGormUserRepository_Get_NoDeletedAtReference_IncludesTrashed(t *testing.T) {
	repo := seedUsersForTrashed(t)
	q := &domainQuery.Query{Page: 1, Limit: 10}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err, "Get is always unscoped: trashed users must be included by default")
	assert.Equal(t, 3, total)
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}
	assert.Contains(t, ids, "trashed-c", "trashed user must be included by default")
}

func TestGormUserRepository_Get_NoDeletedAtFilter_WithSortsAndPagination_IncludesTrashed(t *testing.T) {
	repo := seedUsersForTrashed(t)
	q := &domainQuery.Query{
		Page:  1,
		Limit: 10,
		Sorts: []domainQuery.Sort{
			domainQuery.NewSort("updated_at", domainQuery.SortDesc),
		},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err, "Get with sorts+pagination must still include trashed users")
	assert.Equal(t, 3, total, "should return 3 total users (2 live + 1 trashed)")
	assert.Len(t, users, 3)

	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}
	assert.Contains(t, ids, "trashed-c", "trashed user must be included after sort+pagination")
}

func TestGormUserRepository_Get_FilterDeletedAtBetween_RangeFilterWorks(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithID("recent-trash"), fixtures.WithEmail("rt@x.com")),
		fixtures.NewDomainUser(fixtures.WithID("live-d"), fixtures.WithEmail("liveD@x.com")),
	)
	_, _ = repo.Delete(context.Background(), "recent-trash")
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	tomorrow := now.Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("deleted_at", domainQuery.OperatorBetween, yesterday, tomorrow)},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "recent-trash", users[0].Id)
}

func TestGormUserRepository_Get_SortByDeletedAtDesc_IncludesTrashedAndLive(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithID("trash-old"), fixtures.WithEmail("to@x.com")),
		fixtures.NewDomainUser(fixtures.WithID("trash-new"), fixtures.WithEmail("tn@x.com")),
		fixtures.NewDomainUser(fixtures.WithID("live"), fixtures.WithEmail("live@x.com")),
	)
	_, _ = repo.Delete(context.Background(), "trash-old")
	time.Sleep(10 * time.Millisecond) // ensure trash-new has a strictly later deleted_at
	_, _ = repo.Delete(context.Background(), "trash-new")
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Sorts: []domainQuery.Sort{{Column: "deleted_at", Order: domainQuery.SortDesc}},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err, "sort by deleted_at must auto-unscope")
	assert.Equal(t, 3, total, "all 3 users (2 trashed + 1 live) must be present")
	assert.Len(t, users, 3)

	// Assert relative order of the two trashed rows (NULL position varies between
	// Postgres and SQLite; we only assert the non-null ordering).
	idIndex := map[string]int{}
	for i, u := range users {
		idIndex[u.Id] = i
	}
	assert.Less(t, idIndex["trash-new"], idIndex["trash-old"], "DESC sort: newer trash before older trash")
}

func TestGormUserRepository_Get_SortByUpdatedAt(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("old@x.com")),
	)
	time.Sleep(10 * time.Millisecond)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("new@x.com")),
	)
	q := &domainQuery.Query{
		Sorts: []domainQuery.Sort{{Column: "updated_at", Order: domainQuery.SortDesc}},
	}
	users, _, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "new@x.com", users[0].Email, "newer updated_at should come first with DESC")
	assert.Equal(t, "old@x.com", users[1].Email)
}

func TestGormUserRepository_Get_SortByUpdatedAtAsc(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("old@x.com")),
	)
	time.Sleep(10 * time.Millisecond)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("new@x.com")),
	)
	q := &domainQuery.Query{
		Sorts: []domainQuery.Sort{{Column: "updated_at", Order: domainQuery.SortAsc}},
	}
	users, _, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "old@x.com", users[0].Email, "older updated_at should come first with ASC")
	assert.Equal(t, "new@x.com", users[1].Email)
}

func TestGormUserRepository_Get_FilterRoleAndDeletedAtNotNull_AndedCorrectly(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithID("trash-holder"), fixtures.WithEmail("th@x.com"), fixtures.WithRole(domain.RoleHolder)),
		fixtures.NewDomainUser(fixtures.WithID("trash-issuer"), fixtures.WithEmail("ti@x.com"), fixtures.WithRole(domain.RoleIssuer)),
		fixtures.NewDomainUser(fixtures.WithID("live-holder"), fixtures.WithEmail("lh@x.com"), fixtures.WithRole(domain.RoleHolder)),
	)
	_, _ = repo.Delete(context.Background(), "trash-holder")
	_, _ = repo.Delete(context.Background(), "trash-issuer")
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Filters: []domainQuery.Filter{
			domainQuery.NewFilter("role", domainQuery.OperatorEqual, "holder"),
			domainQuery.NewFilter("deleted_at", domainQuery.OperatorNotNull),
		},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "trash-holder", users[0].Id)
}

func TestGormUserRepository_Get_FilterDeletedAtNotNull_PaginationCountConsistent(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithID("t1"), fixtures.WithEmail("t1@x.com")),
		fixtures.NewDomainUser(fixtures.WithID("t2"), fixtures.WithEmail("t2@x.com")),
		fixtures.NewDomainUser(fixtures.WithID("t3"), fixtures.WithEmail("t3@x.com")),
	)
	_, _ = repo.Delete(context.Background(), "t1", "t2", "t3")
	q := &domainQuery.Query{
		Page: 1, Limit: 1,
		Filters: []domainQuery.Filter{domainQuery.NewFilter("deleted_at", domainQuery.OperatorNotNull)},
	}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 3, total, "total reflects all trashed, not the page slice")
	assert.Len(t, users, 1)
}

func TestGormUserRepository_Restore_Empty(t *testing.T) {
	repo := newRepo(t)
	n, err := repo.Restore(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, 0, n)
}

func TestGormUserRepository_Restore_Single(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("rs1"), fixtures.WithEmail("rs1@x.com"))
	_, _ = repo.Store(context.Background(), u)
	_, _ = repo.Delete(context.Background(), "rs1")

	found, err := repo.FindByIds(context.Background(), "rs1")
	assert.NoError(t, err)
	assert.Len(t, found, 1)
	assert.NotNil(t, found[0].DeletedAt, "user must be trashed before restore")

	n, err := repo.Restore(context.Background(), "rs1")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, n)

	restored, err := repo.FindByIds(context.Background(), "rs1")
	assert.NoError(t, err)
	assert.Len(t, restored, 1)
	assert.Nil(t, restored[0].DeletedAt, "deleted_at must be nil after restore")
}

func TestGormUserRepository_Restore_Batch(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("rb1"), fixtures.WithEmail("rb1@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("rb2"), fixtures.WithEmail("rb2@x.com"))
	u3 := fixtures.NewDomainUser(fixtures.WithID("rb3"), fixtures.WithEmail("rb3@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2, u3)
	_, _ = repo.Delete(context.Background(), "rb1", "rb2", "rb3")

	n, err := repo.Restore(context.Background(), "rb1", "rb2", "rb3")
	assert.NoError(t, err)
	assert.EqualValues(t, 3, n)

	restored, err := repo.FindByIds(context.Background(), "rb1", "rb2", "rb3")
	assert.NoError(t, err)
	assert.Len(t, restored, 3)
	for _, u := range restored {
		assert.Nil(t, u.DeletedAt, "all restored users must have nil deleted_at")
	}
}

func TestGormUserRepository_Restore_PartialBatch(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("rp1"), fixtures.WithEmail("rp1@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("rp2"), fixtures.WithEmail("rp2@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2)
	_, _ = repo.Delete(context.Background(), "rp1")

	n, err := repo.Restore(context.Background(), "rp1", "rp2")
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n, "rows affected counts every matched row, even if deleted_at was already nil")

	restored, err := repo.FindByIds(context.Background(), "rp1", "rp2")
	assert.NoError(t, err)
	assert.Len(t, restored, 2)
	for _, u := range restored {
		assert.Nil(t, u.DeletedAt, "both users must have nil deleted_at after restore")
	}
}

func TestGormUserRepository_Restore_LiveUser_NoChange(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("rl1"), fixtures.WithEmail("rl1@x.com"))
	_, _ = repo.Store(context.Background(), u)

	n, err := repo.Restore(context.Background(), "rl1")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, n, "row matched by WHERE clause counts as affected even if value unchanged")

	found, err := repo.Find(context.Background(), "rl1")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Nil(t, found.DeletedAt)
}

func TestGormUserRepository_Restore_PreservesRole(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("rr1"), fixtures.WithEmail("rr1@x.com"), fixtures.WithRole(domain.RoleAdmin))
	_, _ = repo.Store(context.Background(), u)
	_, _ = repo.Delete(context.Background(), "rr1")

	n, err := repo.Restore(context.Background(), "rr1")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, n)

	restored, err := repo.Find(context.Background(), "rr1")
	assert.NoError(t, err)
	assert.Nil(t, restored.DeletedAt)
	assert.Equal(t, domain.RoleAdmin, restored.Role, "role must be preserved after restore")
}

func TestGormUserGet_FilterAndSortByUnitAndJoinedYear(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithEmail("u1@x.com"), fixtures.WithUnitID("unit-a"), fixtures.WithJoinedYear(2020)),
		fixtures.NewDomainUser(fixtures.WithID("u2"), fixtures.WithEmail("u2@x.com"), fixtures.WithUnitID("unit-b"), fixtures.WithJoinedYear(2021)),
	)
	require.NoError(t, err)

	q := &domainQuery.Query{Filters: []domainQuery.Filter{
		domainQuery.NewFilter("unit_id", domainQuery.OperatorEqual, "unit-a"),
	}}
	got, total, err := repo.Get(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, got, 1)
	assert.Equal(t, "u1", got[0].Id)

	q2 := &domainQuery.Query{Sorts: []domainQuery.Sort{{Column: "joined_year", Order: domainQuery.SortDesc}}}
	got2, _, err := repo.Get(ctx, q2)
	require.NoError(t, err)
	require.Len(t, got2, 2)
	assert.Equal(t, "u2", got2[0].Id)
}
