// Code generated by pggen DO NOT EDIT.

package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/ethanpailes/pgtypes"
	"github.com/opendoor/pggen"
	"github.com/opendoor/pggen/include"
	"github.com/opendoor/pggen/unstable"
)

// PGClient wraps either a 'sql.DB' or a 'sql.Tx'. All pggen-generated
// database access methods for this package are attached to it.
type PGClient struct {
	impl       pgClientImpl
	topLevelDB pggen.DBConn

	errorConverter func(error) error

	// These column indexes are used at run time to enable us to 'SELECT *' against
	// a table that has the same columns in a different order from the ones that we
	// saw in the table we used to generate code. This means that you don't have to worry
	// about migrations merging in a slightly different order than their timestamps have
	// breaking 'SELECT *'.
	rwlockForUser                             sync.RWMutex
	colIdxTabForUser                          []int
	rwlockForGetUsersByNullableNicknameRow    sync.RWMutex
	colIdxTabForGetUsersByNullableNicknameRow []int
}

// bogus usage so we can compile with no tables configured
var _ = sync.RWMutex{}

// NewPGClient creates a new PGClient out of a '*sql.DB' or a
// custom wrapper around a db connection.
//
// If you provide your own wrapper around a '*sql.DB' for logging or
// custom tracing, you MUST forward all calls to an underlying '*sql.DB'
// member of your wrapper.
//
// If the DBConn passed into NewPGClient implements an ErrorConverter
// method which returns a func(error) error, the result of calling the
// ErrorConverter method will be called on every error that the generated
// code returns right before the error is returned. If ErrorConverter
// returns nil or is not present, it will default to the identity function.
func NewPGClient(conn pggen.DBConn) *PGClient {
	client := PGClient{
		topLevelDB: conn,
	}
	client.impl = pgClientImpl{
		db:     conn,
		client: &client,
	}

	// extract the optional error converter routine
	ec, ok := conn.(interface {
		ErrorConverter() func(error) error
	})
	if ok {
		client.errorConverter = ec.ErrorConverter()
	}
	if client.errorConverter == nil {
		client.errorConverter = func(err error) error { return err }
	}

	return &client
}

func (p *PGClient) Handle() pggen.DBHandle {
	return p.topLevelDB
}

func (p *PGClient) BeginTx(ctx context.Context, opts *sql.TxOptions) (*TxPGClient, error) {
	tx, err := p.topLevelDB.BeginTx(ctx, opts)
	if err != nil {
		return nil, p.errorConverter(err)
	}

	return &TxPGClient{
		impl: pgClientImpl{
			db:     tx,
			client: p,
		},
	}, nil
}

func (p *PGClient) Conn(ctx context.Context) (*ConnPGClient, error) {
	conn, err := p.topLevelDB.Conn(ctx)
	if err != nil {
		return nil, p.errorConverter(err)
	}

	return &ConnPGClient{impl: pgClientImpl{db: conn, client: p}}, nil
}

// A postgres client that operates within a transaction. Supports all the same
// generated methods that PGClient does.
type TxPGClient struct {
	impl pgClientImpl
}

func (tx *TxPGClient) Handle() pggen.DBHandle {
	return tx.impl.db.(*sql.Tx)
}

func (tx *TxPGClient) Rollback() error {
	return tx.impl.db.(*sql.Tx).Rollback()
}

func (tx *TxPGClient) Commit() error {
	return tx.impl.db.(*sql.Tx).Commit()
}

type ConnPGClient struct {
	impl pgClientImpl
}

func (conn *ConnPGClient) Close() error {
	return conn.impl.db.(*sql.Conn).Close()
}

func (conn *ConnPGClient) Handle() pggen.DBHandle {
	return conn.impl.db
}

// A database client that can wrap either a direct database connection or a transaction
type pgClientImpl struct {
	db pggen.DBHandle
	// a reference back to the owning PGClient so we can always get at the resolver tables
	client *PGClient
}

func (p *PGClient) GetUser(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*User, error) {
	return p.impl.getUser(ctx, id)
}
func (tx *TxPGClient) GetUser(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*User, error) {
	return tx.impl.getUser(ctx, id)
}
func (conn *ConnPGClient) GetUser(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*User, error) {
	return conn.impl.getUser(ctx, id)
}
func (p *pgClientImpl) getUser(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*User, error) {
	values, err := p.listUser(ctx, []int64{id}, true /* isGet */)
	if err != nil {
		return nil, err
	}

	// ListUser always returns the same number of records as were
	// requested, so this is safe.
	return &values[0], err
}

func (p *PGClient) ListUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []User, err error) {
	return p.impl.listUser(ctx, ids, false /* isGet */)
}
func (tx *TxPGClient) ListUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []User, err error) {
	return tx.impl.listUser(ctx, ids, false /* isGet */)
}
func (conn *ConnPGClient) ListUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []User, err error) {
	return conn.impl.listUser(ctx, ids, false /* isGet */)
}
func (p *pgClientImpl) listUser(
	ctx context.Context,
	ids []int64,
	isGet bool,
	opts ...pggen.ListOpt,
) (ret []User, err error) {
	if len(ids) == 0 {
		return []User{}, nil
	}

	rows, err := p.queryContext(
		ctx,
		`SELECT * FROM users WHERE "id" = ANY($1)`,
		pgtypes.Array(ids),
	)
	if err != nil {
		return nil, p.client.errorConverter(err)
	}
	defer func() {
		if err == nil {
			err = rows.Close()
			if err != nil {
				ret = nil
				err = p.client.errorConverter(err)
			}
		} else {
			rowErr := rows.Close()
			if rowErr != nil {
				err = p.client.errorConverter(fmt.Errorf("%s AND %s", err.Error(), rowErr.Error()))
			}
		}
	}()

	ret = make([]User, 0, len(ids))
	for rows.Next() {
		var value User
		err = value.Scan(ctx, p.client, rows)
		if err != nil {
			return nil, p.client.errorConverter(err)
		}
		ret = append(ret, value)
	}

	if len(ret) != len(ids) {
		if isGet {
			return nil, p.client.errorConverter(&unstable.NotFoundError{
				Msg: "GetUser: record not found",
			})
		} else {
			return nil, p.client.errorConverter(&unstable.NotFoundError{
				Msg: fmt.Sprintf(
					"ListUser: asked for %d records, found %d",
					len(ids),
					len(ret),
				),
			})
		}
	}

	return ret, nil
}

// Insert a User into the database. Returns the primary
// key of the inserted row.
func (p *PGClient) InsertUser(
	ctx context.Context,
	value *User,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return p.impl.insertUser(ctx, value, opts...)
}

// Insert a User into the database. Returns the primary
// key of the inserted row.
func (tx *TxPGClient) InsertUser(
	ctx context.Context,
	value *User,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return tx.impl.insertUser(ctx, value, opts...)
}

// Insert a User into the database. Returns the primary
// key of the inserted row.
func (conn *ConnPGClient) InsertUser(
	ctx context.Context,
	value *User,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return conn.impl.insertUser(ctx, value, opts...)
}

// Insert a User into the database. Returns the primary
// key of the inserted row.
func (p *pgClientImpl) insertUser(
	ctx context.Context,
	value *User,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	var ids []int64
	ids, err = p.bulkInsertUser(ctx, []User{*value}, opts...)
	if err != nil {
		return ret, p.client.errorConverter(err)
	}

	if len(ids) != 1 {
		return ret, p.client.errorConverter(fmt.Errorf("inserting a User: %d ids (expected 1)", len(ids)))
	}

	ret = ids[0]
	return
}

// Insert a list of User. Returns a list of the primary keys of
// the inserted rows.
func (p *PGClient) BulkInsertUser(
	ctx context.Context,
	values []User,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return p.impl.bulkInsertUser(ctx, values, opts...)
}

// Insert a list of User. Returns a list of the primary keys of
// the inserted rows.
func (tx *TxPGClient) BulkInsertUser(
	ctx context.Context,
	values []User,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return tx.impl.bulkInsertUser(ctx, values, opts...)
}

// Insert a list of User. Returns a list of the primary keys of
// the inserted rows.
func (conn *ConnPGClient) BulkInsertUser(
	ctx context.Context,
	values []User,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return conn.impl.bulkInsertUser(ctx, values, opts...)
}

// Insert a list of User. Returns a list of the primary keys of
// the inserted rows.
func (p *pgClientImpl) bulkInsertUser(
	ctx context.Context,
	values []User,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	if len(values) == 0 {
		return []int64{}, nil
	}

	opt := pggen.InsertOptions{}
	for _, o := range opts {
		o(&opt)
	}

	defaultFields := opt.DefaultFields.Intersection(defaultableColsForUser)
	args := make([]interface{}, 0, 3*len(values))
	for _, v := range values {
		if opt.UsePkey && !defaultFields.Test(UserIdFieldIndex) {
			args = append(args, v.Id)
		}
		if !defaultFields.Test(UserEmailFieldIndex) {
			args = append(args, v.Email)
		}
		if !defaultFields.Test(UserNicknameFieldIndex) {
			args = append(args, v.Nickname)
		}
	}

	bulkInsertQuery := genBulkInsertStmt(
		`users`,
		fieldsForUser,
		len(values),
		"id",
		opt.UsePkey,
		defaultFields,
	)

	rows, err := p.queryContext(ctx, bulkInsertQuery, args...)
	if err != nil {
		return nil, p.client.errorConverter(err)
	}
	defer rows.Close()

	ids := make([]int64, 0, len(values))
	for rows.Next() {
		var id int64
		err = rows.Scan(&(id))
		if err != nil {
			return nil, p.client.errorConverter(err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// bit indicies for 'fieldMask' parameters
const (
	UserIdFieldIndex       int = 0
	UserEmailFieldIndex    int = 1
	UserNicknameFieldIndex int = 2
	UserMaxFieldIndex      int = (3 - 1)
)

// A field set saying that all fields in User should be updated.
// For use as a 'fieldMask' parameter
var UserAllFields pggen.FieldSet = pggen.NewFieldSetFilled(3)

var defaultableColsForUser = func() pggen.FieldSet {
	fs := pggen.NewFieldSet(UserMaxFieldIndex)
	fs.Set(UserIdFieldIndex, true)
	return fs
}()

var fieldsForUser []fieldNameAndIdx = []fieldNameAndIdx{
	{name: `id`, idx: UserIdFieldIndex},
	{name: `email`, idx: UserEmailFieldIndex},
	{name: `nickname`, idx: UserNicknameFieldIndex},
}

// Update a User. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (p *PGClient) UpdateUser(
	ctx context.Context,
	value *User,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return p.impl.updateUser(ctx, value, fieldMask, opts...)
}

// Update a User. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (tx *TxPGClient) UpdateUser(
	ctx context.Context,
	value *User,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return tx.impl.updateUser(ctx, value, fieldMask, opts...)
}

// Update a User. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (conn *ConnPGClient) UpdateUser(
	ctx context.Context,
	value *User,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return conn.impl.updateUser(ctx, value, fieldMask, opts...)
}
func (p *pgClientImpl) updateUser(
	ctx context.Context,
	value *User,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {

	opt := pggen.UpdateOptions{}
	for _, o := range opts {
		o(&opt)
	}

	if !fieldMask.Test(UserIdFieldIndex) {
		return ret, p.client.errorConverter(fmt.Errorf(`primary key required for updates to 'users'`))
	}

	updateStmt := genUpdateStmt(
		`users`,
		"id",
		fieldsForUser,
		fieldMask,
		"id",
	)

	args := make([]interface{}, 0, 3)
	if fieldMask.Test(UserIdFieldIndex) {
		args = append(args, value.Id)
	}
	if fieldMask.Test(UserEmailFieldIndex) {
		args = append(args, value.Email)
	}
	if fieldMask.Test(UserNicknameFieldIndex) {
		args = append(args, value.Nickname)
	}

	// add the primary key arg for the WHERE condition
	args = append(args, value.Id)

	var id int64
	err = p.db.QueryRowContext(ctx, updateStmt, args...).
		Scan(&(id))
	if err != nil {
		return ret, p.client.errorConverter(err)
	}

	return id, nil
}

// Upsert a User value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (p *PGClient) UpsertUser(
	ctx context.Context,
	value *User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = p.impl.bulkUpsertUser(ctx, []User{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a User value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (tx *TxPGClient) UpsertUser(
	ctx context.Context,
	value *User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = tx.impl.bulkUpsertUser(ctx, []User{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a User value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (conn *ConnPGClient) UpsertUser(
	ctx context.Context,
	value *User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = conn.impl.bulkUpsertUser(ctx, []User{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a set of User values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (p *PGClient) BulkUpsertUser(
	ctx context.Context,
	values []User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return p.impl.bulkUpsertUser(ctx, values, constraintNames, fieldMask, opts...)
}

// Upsert a set of User values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (tx *TxPGClient) BulkUpsertUser(
	ctx context.Context,
	values []User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return tx.impl.bulkUpsertUser(ctx, values, constraintNames, fieldMask, opts...)
}

// Upsert a set of User values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (conn *ConnPGClient) BulkUpsertUser(
	ctx context.Context,
	values []User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return conn.impl.bulkUpsertUser(ctx, values, constraintNames, fieldMask, opts...)
}
func (p *pgClientImpl) bulkUpsertUser(
	ctx context.Context,
	values []User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) ([]int64, error) {
	if len(values) == 0 {
		return []int64{}, nil
	}

	options := pggen.UpsertOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if constraintNames == nil || len(constraintNames) == 0 {
		constraintNames = []string{`id`}
	}

	defaultFields := options.DefaultFields.Intersection(defaultableColsForUser)
	var stmt strings.Builder
	genInsertCommon(
		&stmt,
		`users`,
		fieldsForUser,
		len(values),
		`id`,
		options.UsePkey,
		defaultFields,
	)

	setBits := fieldMask.CountSetBits()
	hasConflictAction := setBits > 1 ||
		(setBits == 1 && fieldMask.Test(UserIdFieldIndex) && options.UsePkey) ||
		(setBits == 1 && !fieldMask.Test(UserIdFieldIndex))

	if hasConflictAction {
		stmt.WriteString("ON CONFLICT (")
		stmt.WriteString(strings.Join(constraintNames, ","))
		stmt.WriteString(") DO UPDATE SET ")

		updateCols := make([]string, 0, 3)
		updateExprs := make([]string, 0, 3)
		if options.UsePkey {
			updateCols = append(updateCols, `id`)
			updateExprs = append(updateExprs, `excluded.id`)
		}
		if fieldMask.Test(UserEmailFieldIndex) {
			updateCols = append(updateCols, `email`)
			updateExprs = append(updateExprs, `excluded.email`)
		}
		if fieldMask.Test(UserNicknameFieldIndex) {
			updateCols = append(updateCols, `nickname`)
			updateExprs = append(updateExprs, `excluded.nickname`)
		}
		if len(updateCols) > 1 {
			stmt.WriteRune('(')
		}
		stmt.WriteString(strings.Join(updateCols, ","))
		if len(updateCols) > 1 {
			stmt.WriteRune(')')
		}
		stmt.WriteString(" = ")
		if len(updateCols) > 1 {
			stmt.WriteRune('(')
		}
		stmt.WriteString(strings.Join(updateExprs, ","))
		if len(updateCols) > 1 {
			stmt.WriteRune(')')
		}
	} else {
		stmt.WriteString("ON CONFLICT DO NOTHING")
	}

	stmt.WriteString(` RETURNING "id"`)

	args := make([]interface{}, 0, 3*len(values))
	for _, v := range values {
		if options.UsePkey && !defaultFields.Test(UserIdFieldIndex) {
			args = append(args, v.Id)
		}
		if !defaultFields.Test(UserEmailFieldIndex) {
			args = append(args, v.Email)
		}
		if !defaultFields.Test(UserNicknameFieldIndex) {
			args = append(args, v.Nickname)
		}
	}

	rows, err := p.queryContext(ctx, stmt.String(), args...)
	if err != nil {
		return nil, p.client.errorConverter(err)
	}
	defer rows.Close()

	ids := make([]int64, 0, len(values))
	for rows.Next() {
		var id int64
		err = rows.Scan(&(id))
		if err != nil {
			return nil, p.client.errorConverter(err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (p *PGClient) DeleteUser(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDeleteUser(ctx, []int64{id}, opts...)
}
func (tx *TxPGClient) DeleteUser(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDeleteUser(ctx, []int64{id}, opts...)
}
func (conn *ConnPGClient) DeleteUser(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return conn.impl.bulkDeleteUser(ctx, []int64{id}, opts...)
}

func (p *PGClient) BulkDeleteUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDeleteUser(ctx, ids, opts...)
}
func (tx *TxPGClient) BulkDeleteUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDeleteUser(ctx, ids, opts...)
}
func (conn *ConnPGClient) BulkDeleteUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return conn.impl.bulkDeleteUser(ctx, ids, opts...)
}
func (p *pgClientImpl) bulkDeleteUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	if len(ids) == 0 {
		return nil
	}

	options := pggen.DeleteOptions{}
	for _, o := range opts {
		o(&options)
	}
	res, err := p.db.ExecContext(
		ctx,
		`DELETE FROM users WHERE "id" = ANY($1)`,
		pgtypes.Array(ids),
	)
	if err != nil {
		return p.client.errorConverter(err)
	}

	nrows, err := res.RowsAffected()
	if err != nil {
		return p.client.errorConverter(err)
	}

	if nrows != int64(len(ids)) {
		return p.client.errorConverter(fmt.Errorf(
			"BulkDeleteUser: %d rows deleted, expected %d",
			nrows,
			len(ids),
		))
	}

	return err
}

var UserAllIncludes *include.Spec = include.Must(include.Parse(
	`users`,
))

func (p *PGClient) UserFillIncludes(
	ctx context.Context,
	rec *User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.privateUserBulkFillIncludes(ctx, []*User{rec}, includes)
}
func (tx *TxPGClient) UserFillIncludes(
	ctx context.Context,
	rec *User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.privateUserBulkFillIncludes(ctx, []*User{rec}, includes)
}
func (conn *ConnPGClient) UserFillIncludes(
	ctx context.Context,
	rec *User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return conn.impl.privateUserBulkFillIncludes(ctx, []*User{rec}, includes)
}

func (p *PGClient) UserBulkFillIncludes(
	ctx context.Context,
	recs []*User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.privateUserBulkFillIncludes(ctx, recs, includes)
}
func (tx *TxPGClient) UserBulkFillIncludes(
	ctx context.Context,
	recs []*User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.privateUserBulkFillIncludes(ctx, recs, includes)
}
func (conn *ConnPGClient) UserBulkFillIncludes(
	ctx context.Context,
	recs []*User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return conn.impl.privateUserBulkFillIncludes(ctx, recs, includes)
}
func (p *pgClientImpl) privateUserBulkFillIncludes(
	ctx context.Context,
	recs []*User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	loadedRecordTab := map[string]interface{}{}

	return p.implUserBulkFillIncludes(ctx, recs, includes, loadedRecordTab)
}

func (p *pgClientImpl) implUserBulkFillIncludes(
	ctx context.Context,
	recs []*User,
	includes *include.Spec,
	loadedRecordTab map[string]interface{},
) (err error) {
	if includes.TableName != `users` {
		return p.client.errorConverter(fmt.Errorf(
			`expected includes for 'users', got '%s'`,
			includes.TableName,
		))
	}

	loadedTab, inMap := loadedRecordTab[`users`]
	if inMap {
		idToRecord := loadedTab.(map[int64]*User)
		for _, r := range recs {
			_, alreadyLoaded := idToRecord[r.Id]
			if !alreadyLoaded {
				idToRecord[r.Id] = r
			}
		}
	} else {
		idToRecord := make(map[int64]*User, len(recs))
		for _, r := range recs {
			idToRecord[r.Id] = r
		}
		loadedRecordTab[`users`] = idToRecord
	}

	return
}

//     This query looks up users by their nickname, even if that nickname is NULL.
//
//     Note the funny `nickname = $1 OR (nickname IS NULL AND $1 IS NULL)` construct.
//     This is a common pattern when querying based on possibly-null parameters. The
//     reason for this has to do with SQL's trinary logic and null propigation. In the
//     context of of most programming languages nulls mean something like "a reference to
//     nothing", but in SQL it is better to think of NULL as meaning "unknown".
//     What is `nickname = UNKNOWN`? Well, we don't know what is on the rhs, so the whole
//     thing is `UNKNOWN`. What about `UNKNOWN OR true`? Well, all we need to know is that
//     one side of the OR is true in order for the whole thing to be true, so the whole
//     expression is `true`.
//
//     If we just wrote `WHERE nickname = $1` as would make sense in most programming
//     languages, we would end up with `WHERE UNKNOWN` when `$1` is NULL, and SQL
//     will only return queries where it knows for sure that the WHERE condition is
//     true, so we would never be able to return any results when `$1` is NULL. If that
//     was the case there would be no point in generating code for nullable arguments in
//     the first place.
func (p *PGClient) GetUsersByNullableNickname(
	ctx context.Context,
	arg1 *string,
) (ret []User, err error) {
	return p.impl.GetUsersByNullableNickname(
		ctx,
		arg1,
	)
}

//     This query looks up users by their nickname, even if that nickname is NULL.
//
//     Note the funny `nickname = $1 OR (nickname IS NULL AND $1 IS NULL)` construct.
//     This is a common pattern when querying based on possibly-null parameters. The
//     reason for this has to do with SQL's trinary logic and null propigation. In the
//     context of of most programming languages nulls mean something like "a reference to
//     nothing", but in SQL it is better to think of NULL as meaning "unknown".
//     What is `nickname = UNKNOWN`? Well, we don't know what is on the rhs, so the whole
//     thing is `UNKNOWN`. What about `UNKNOWN OR true`? Well, all we need to know is that
//     one side of the OR is true in order for the whole thing to be true, so the whole
//     expression is `true`.
//
//     If we just wrote `WHERE nickname = $1` as would make sense in most programming
//     languages, we would end up with `WHERE UNKNOWN` when `$1` is NULL, and SQL
//     will only return queries where it knows for sure that the WHERE condition is
//     true, so we would never be able to return any results when `$1` is NULL. If that
//     was the case there would be no point in generating code for nullable arguments in
//     the first place.
func (tx *TxPGClient) GetUsersByNullableNickname(
	ctx context.Context,
	arg1 *string,
) (ret []User, err error) {
	return tx.impl.GetUsersByNullableNickname(
		ctx,
		arg1,
	)
}

//     This query looks up users by their nickname, even if that nickname is NULL.
//
//     Note the funny `nickname = $1 OR (nickname IS NULL AND $1 IS NULL)` construct.
//     This is a common pattern when querying based on possibly-null parameters. The
//     reason for this has to do with SQL's trinary logic and null propigation. In the
//     context of of most programming languages nulls mean something like "a reference to
//     nothing", but in SQL it is better to think of NULL as meaning "unknown".
//     What is `nickname = UNKNOWN`? Well, we don't know what is on the rhs, so the whole
//     thing is `UNKNOWN`. What about `UNKNOWN OR true`? Well, all we need to know is that
//     one side of the OR is true in order for the whole thing to be true, so the whole
//     expression is `true`.
//
//     If we just wrote `WHERE nickname = $1` as would make sense in most programming
//     languages, we would end up with `WHERE UNKNOWN` when `$1` is NULL, and SQL
//     will only return queries where it knows for sure that the WHERE condition is
//     true, so we would never be able to return any results when `$1` is NULL. If that
//     was the case there would be no point in generating code for nullable arguments in
//     the first place.
func (conn *ConnPGClient) GetUsersByNullableNickname(
	ctx context.Context,
	arg1 *string,
) (ret []User, err error) {
	return conn.impl.GetUsersByNullableNickname(
		ctx,
		arg1,
	)
}
func (p *pgClientImpl) GetUsersByNullableNickname(
	ctx context.Context,
	arg1 *string,
) (ret []User, err error) {
	ret = []User{}

	var rows *sql.Rows
	rows, err = p.GetUsersByNullableNicknameQuery(
		ctx,
		arg1,
	)
	if err != nil {
		return nil, p.client.errorConverter(err)
	}
	defer func() {
		if err == nil {
			err = rows.Close()
			if err != nil {
				ret = nil
				err = p.client.errorConverter(err)
			}
		} else {
			rowErr := rows.Close()
			if rowErr != nil {
				err = p.client.errorConverter(fmt.Errorf("%s AND %s", err.Error(), rowErr.Error()))
			}
		}
	}()

	for rows.Next() {
		var row User
		err = row.Scan(ctx, p.client, rows)
		ret = append(ret, row)
	}

	return
}

//     This query looks up users by their nickname, even if that nickname is NULL.
//
//     Note the funny `nickname = $1 OR (nickname IS NULL AND $1 IS NULL)` construct.
//     This is a common pattern when querying based on possibly-null parameters. The
//     reason for this has to do with SQL's trinary logic and null propigation. In the
//     context of of most programming languages nulls mean something like "a reference to
//     nothing", but in SQL it is better to think of NULL as meaning "unknown".
//     What is `nickname = UNKNOWN`? Well, we don't know what is on the rhs, so the whole
//     thing is `UNKNOWN`. What about `UNKNOWN OR true`? Well, all we need to know is that
//     one side of the OR is true in order for the whole thing to be true, so the whole
//     expression is `true`.
//
//     If we just wrote `WHERE nickname = $1` as would make sense in most programming
//     languages, we would end up with `WHERE UNKNOWN` when `$1` is NULL, and SQL
//     will only return queries where it knows for sure that the WHERE condition is
//     true, so we would never be able to return any results when `$1` is NULL. If that
//     was the case there would be no point in generating code for nullable arguments in
//     the first place.
func (p *PGClient) GetUsersByNullableNicknameQuery(
	ctx context.Context,
	arg1 *string,
) (*sql.Rows, error) {
	return p.impl.GetUsersByNullableNicknameQuery(
		ctx,
		arg1,
	)
}

//     This query looks up users by their nickname, even if that nickname is NULL.
//
//     Note the funny `nickname = $1 OR (nickname IS NULL AND $1 IS NULL)` construct.
//     This is a common pattern when querying based on possibly-null parameters. The
//     reason for this has to do with SQL's trinary logic and null propigation. In the
//     context of of most programming languages nulls mean something like "a reference to
//     nothing", but in SQL it is better to think of NULL as meaning "unknown".
//     What is `nickname = UNKNOWN`? Well, we don't know what is on the rhs, so the whole
//     thing is `UNKNOWN`. What about `UNKNOWN OR true`? Well, all we need to know is that
//     one side of the OR is true in order for the whole thing to be true, so the whole
//     expression is `true`.
//
//     If we just wrote `WHERE nickname = $1` as would make sense in most programming
//     languages, we would end up with `WHERE UNKNOWN` when `$1` is NULL, and SQL
//     will only return queries where it knows for sure that the WHERE condition is
//     true, so we would never be able to return any results when `$1` is NULL. If that
//     was the case there would be no point in generating code for nullable arguments in
//     the first place.
func (tx *TxPGClient) GetUsersByNullableNicknameQuery(
	ctx context.Context,
	arg1 *string,
) (*sql.Rows, error) {
	return tx.impl.GetUsersByNullableNicknameQuery(
		ctx,
		arg1,
	)
}

//     This query looks up users by their nickname, even if that nickname is NULL.
//
//     Note the funny `nickname = $1 OR (nickname IS NULL AND $1 IS NULL)` construct.
//     This is a common pattern when querying based on possibly-null parameters. The
//     reason for this has to do with SQL's trinary logic and null propigation. In the
//     context of of most programming languages nulls mean something like "a reference to
//     nothing", but in SQL it is better to think of NULL as meaning "unknown".
//     What is `nickname = UNKNOWN`? Well, we don't know what is on the rhs, so the whole
//     thing is `UNKNOWN`. What about `UNKNOWN OR true`? Well, all we need to know is that
//     one side of the OR is true in order for the whole thing to be true, so the whole
//     expression is `true`.
//
//     If we just wrote `WHERE nickname = $1` as would make sense in most programming
//     languages, we would end up with `WHERE UNKNOWN` when `$1` is NULL, and SQL
//     will only return queries where it knows for sure that the WHERE condition is
//     true, so we would never be able to return any results when `$1` is NULL. If that
//     was the case there would be no point in generating code for nullable arguments in
//     the first place.
func (conn *ConnPGClient) GetUsersByNullableNicknameQuery(
	ctx context.Context,
	arg1 *string,
) (*sql.Rows, error) {
	return conn.impl.GetUsersByNullableNicknameQuery(
		ctx,
		arg1,
	)
}
func (p *pgClientImpl) GetUsersByNullableNicknameQuery(
	ctx context.Context,
	arg1 *string,
) (*sql.Rows, error) {
	return p.queryContext(
		ctx,
		`SELECT * FROM users WHERE nickname = $1 OR (nickname IS NULL AND $1 IS NULL)`,
		arg1,
	)
}

type DBQueries interface {
	//
	// automatic CRUD methods
	//

	// User methods
	GetUser(ctx context.Context, id int64, opts ...pggen.GetOpt) (*User, error)
	ListUser(ctx context.Context, ids []int64, opts ...pggen.ListOpt) ([]User, error)
	InsertUser(ctx context.Context, value *User, opts ...pggen.InsertOpt) (int64, error)
	BulkInsertUser(ctx context.Context, values []User, opts ...pggen.InsertOpt) ([]int64, error)
	UpdateUser(ctx context.Context, value *User, fieldMask pggen.FieldSet, opts ...pggen.UpdateOpt) (ret int64, err error)
	UpsertUser(ctx context.Context, value *User, constraintNames []string, fieldMask pggen.FieldSet, opts ...pggen.UpsertOpt) (int64, error)
	BulkUpsertUser(ctx context.Context, values []User, constraintNames []string, fieldMask pggen.FieldSet, opts ...pggen.UpsertOpt) ([]int64, error)
	DeleteUser(ctx context.Context, id int64, opts ...pggen.DeleteOpt) error
	BulkDeleteUser(ctx context.Context, ids []int64, opts ...pggen.DeleteOpt) error
	UserFillIncludes(ctx context.Context, rec *User, includes *include.Spec, opts ...pggen.IncludeOpt) error
	UserBulkFillIncludes(ctx context.Context, recs []*User, includes *include.Spec, opts ...pggen.IncludeOpt) error

	//
	// query methods
	//

	// GetUsersByNullableNickname query
	GetUsersByNullableNickname(
		ctx context.Context,
		arg1 *string,
	) ([]User, error)
	GetUsersByNullableNicknameQuery(
		ctx context.Context,
		arg1 string,
	) (*sql.Rows, error)

	//
	// stored function methods
	//

	//
	// stmt methods
	//

}

type User struct {
	Id       int64   `gorm:"column:id;is_primary"`
	Email    string  `gorm:"column:email"`
	Nickname *string `gorm:"column:nickname"`
}

func (r *User) Scan(ctx context.Context, client *PGClient, rs *sql.Rows) error {
	client.rwlockForUser.RLock()
	if client.colIdxTabForUser == nil {
		client.rwlockForUser.RUnlock() // release the lock to allow the write lock to be aquired
		err := client.fillColPosTab(
			ctx,
			genTimeColIdxTabForUser,
			&client.rwlockForUser,
			rs,
			&client.colIdxTabForUser,
		)
		if err != nil {
			return err
		}
		client.rwlockForUser.RLock() // get the lock back for the rest of the routine
	}

	var nullableTgts nullableScanTgtsForUser

	scanTgts := make([]interface{}, len(client.colIdxTabForUser))
	for runIdx, genIdx := range client.colIdxTabForUser {
		if genIdx == -1 {
			scanTgts[runIdx] = &pggenSinkScanner{}
		} else {
			scanTgts[runIdx] = scannerTabForUser[genIdx](r, &nullableTgts)
		}
	}
	client.rwlockForUser.RUnlock() // we are now done referencing the idx tab in the happy path

	err := rs.Scan(scanTgts...)
	if err != nil {
		// The database schema may have been changed out from under us, let's
		// check to see if we just need to update our column index tables and retry.
		colNames, colsErr := rs.Columns()
		if colsErr != nil {
			return fmt.Errorf("pggen: checking column names: %s", colsErr.Error())
		}
		client.rwlockForUser.RLock()
		if len(client.colIdxTabForUser) != len(colNames) {
			client.rwlockForUser.RUnlock() // release the lock to allow the write lock to be aquired
			err = client.fillColPosTab(
				ctx,
				genTimeColIdxTabForUser,
				&client.rwlockForUser,
				rs,
				&client.colIdxTabForUser,
			)
			if err != nil {
				return err
			}

			return r.Scan(ctx, client, rs)
		} else {
			client.rwlockForUser.RUnlock()
			return err
		}
	}
	r.Nickname = convertNullString(nullableTgts.scanNickname)

	return nil
}

type nullableScanTgtsForUser struct {
	scanNickname sql.NullString
}

// a table mapping codegen-time col indicies to functions returning a scanner for the
// field that was at that column index at codegen-time.
var scannerTabForUser = [...]func(*User, *nullableScanTgtsForUser) interface{}{
	func(
		r *User,
		nullableTgts *nullableScanTgtsForUser,
	) interface{} {
		return &(r.Id)
	},
	func(
		r *User,
		nullableTgts *nullableScanTgtsForUser,
	) interface{} {
		return &(r.Email)
	},
	func(
		r *User,
		nullableTgts *nullableScanTgtsForUser,
	) interface{} {
		return &(nullableTgts.scanNickname)
	},
}

var genTimeColIdxTabForUser map[string]int = map[string]int{
	`id`:       0,
	`email`:    1,
	`nickname`: 2,
}
var _ = unstable.NotFoundError{}
