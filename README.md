# pggen

`pggen` consists of 3 main packages:
  - `github.com/opendoor/pggen/cmd/pggen` is a command line tool to be
     invoked in `go:generate` directives.
  - `github.com/opendoor/pggen/gen` is the back end for the
    `github.com/opendoor/pggen/cmd/pggen` tool. All the command line
    flags exposed by the command line tool have corrisponding fields in
    the `Config` struct exposed by this package, so pggen can be used
    as a library as well as a command line tool.
  - `github.com/opendoor/pggen` and its subpackages (besides `cmd` and `gen`)
    contain common types and utilities that need to be shared between client code and the
    code generated by `pggen`.

# Development

See [HACKING.md](HACKING.md) for information about working on the pggen
codebase.

# Using `pggen`

The `pggen` command line tool has a fairly simple interface. You need
to tell pggen how to connect to the postgres database that you want to
generate code for either by passing the `--connection-string` option or
by setting the `DB_URL` environment variable. It is a good idea to
tell pggen where to put the code that it generates via the `-o` option,
but you can also just accept the default output name. Finally, you must
point `pggen` at a `toml` file which contains the bulk of the configuration
telling `pggen` what part of the database schema to generate code for.

`pggen` will not generate code for any database object unless it is explicitly
asked do so so via an entry in the config file. Often, configuring `pggen`
to generate code for an object is as simple as adding that object's name
to the config file and letting `pggen` figure out the rest, but there are
finer grained knobs if you want more control.

## Configuration

`pggen` is configured with a `toml` file. Some of the configuration options have already
been mentioned in this document, but the most complete source of documentation
on configuration is the comments in [`gen/internal/config/config.go`](gen/internal/config/config.go).
An example file can be found at [`cmd/pggen/test/models/pggen.toml`](cmd/pggen/test/models/pggen.toml).

## [Examples](./examples)

The [examples directory](./examples) contains usage examples and common patterns.

## Features

`pggen` offers two main features: automatic generation of shims wrapping
SQL queries and automatic generation of go structs from SQL tables.

### Query Shims

`pggen` knows how to infer the return and argument types of queries, so all
you have to write is the SQL that you want to execute using standard postgres
$N placeholder syntax for parameters and pggen will generate simple go
wrapper functions that perform all the boilerplate needed to call the
query from go. `pggen` will automatically generate a struct to contain
the result rows that the query returns, though if you want to re-use
a return type between queries you can do so by providing a return
type name.

If you have the following entry in your toml file

```toml
[[query]]
    name = "GetIdAndCreated"
    body = '''
    SELECT id, created_at
    FROM foo
    WHERE ID = $1
    ORDER BY created_at
    '''
```

and the following DDL to define your database schema

```sql
CREATE TABLE foo (
    id SERIAL PRIMARY KEY NOT NULL,
    created_at TIMESTAMP,
    ...
);
```

`pggen` will generate a return type

```golang
type GetIdAndCreatedRow struct {
    Id *int64
    CreatedAt *time.Time
}
```

for you. `GetIdAndCreated` will have a `Scan` method which accepts a
`*sql.Rows` as and argument. `pggen` will also generate two functions
`GetIdAndCreated` and `GetIdAndCreatedQuery`.

`GetIDAndCreated` should be your go-to method for invoking this query.
It accepts a `context.Context` and all the arguments to the query, in this case
just a single `int64` argument, and returns a slice of `GetIdAndCreatedRow`s along
with a possible error. This essentially turns an SQL query into a type safe RPC call
to the database.

Sometimes you don't want to load all of the results of a query into memory at once,
in which case `GetIdAndCreatedQuery` is useful. It accepts the same arguments as
`GetIdAndCreated` and returns `(*sql.Rows, error)`. This isn't much higher level than
just placing the SQL call directly, but you still retain the benefit of having type
safe query parameters. Once you have the `*sql.Rows` in hand, you can make use of
the `Scan` method on `GetIdAndCreatedRow` to lazily load query results in a loop.

#### Named Return Types

If you don't provide a name for your return type `pggen` is happy to
make one up, but if you want to override the fairly uninspired name
that `pggen` will come up with, you can do so. This feature is also
the key to processing the result types of multiple queries with the
same code. In the above example, this would allow you to override
the name of `GetIdAndCreatedRow`.

#### Not Null Fields

Postgres does not perform inference about the nullability of the fields
returned via a query, so by default `pggen` will generate boxed fields
for the return struct. If you know for sure that certain query result
fields cannot ever be null, you may use the `not_null_fields`
configuration option to tell `pggen` not to box the fields in question.
If you are re-using a return type between queries, be sure to apply this
flag consistently when dealing with columns that appear in multiple `pggen`
queries, as both the field names and their nullability values must match up
in order for the generated type to be reused.

Instead of providing a list of `NOT NULL` fields, you can also provide a more
compact specification of the nullability of the result rows with the `null_flags`
configuration option. In general it is better to use the `not_null_fields`
configuration option, as it is more explicit, but `null_flags` can be more
useful when a return column does not have a clear name. The value of the null
flags configuration option, if it is provided, should be a string that is exactly
as long as the number of fields that the query returns. For each field in the return
type, the character at the corresponding position in the null flags string
indicates the nullability of the field, with '-' meaning that the field is
NOT NULL and 'n' indicating that the field is nullable.

When returning a type generated from a table, you do not need to set the
null flags, as `pggen` will automatically infer the nullness of the fields
from the nullness of the fields in the table.

If you knew for a fact that the `id` and `created_at` fields could not be null
in the above example, you could modify your toml entry to read

```toml
[[query]]
    name = "GetIdAndCreated"
    body = '''
    SELECT id, created_at
    FROM foo
    WHERE ID = $1
    ORDER BY created_at
    '''
    not_null_fields = ["id", "created_at"]
```

or equivalently

```toml
[[query]]
    name = "GetIdAndCreated"
    body = '''
    SELECT id, created_at
    FROM foo
    WHERE ID = $1
    ORDER BY created_at
    '''
    null_flags = "--"
```

which would cause the result type

```golang
type GetIdAndCreatedRow struct {
	Id int64 `gorm:"column:id;is_primary"`
	CreatedAt time.Time `gorm:"column:created_at"`
}
```

to be generated. Note the fact that the fields are no longer boxed.

### Model Structs

`pggen` translates table definitions into golang structs along with
a stable of common CRUD operations for working with those structs.
In addition to the provided CRUD operations, you can use the model
structs generated by `pggen` as return values from your own custom
queries. You can also easily use your own custom dynamically
generated SQL to produce model structs by making use of the `Scan` method
attached to all of them.

#### Generated Code for Tables

The generated struct for a postgres table is very similar to the generated
struct for a query return value. Postgres does expose the nullability of table
columns, so you don't have to worry about explicitly setting null flags
for a table.

If you had the DDL

```sql
CREATE TABLE small_entities (
    id SERIAL PRIMARY KEY NOT NULL,
    anint integer NOT NULL
);

CREATE TABLE attachments (
    id UUID PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    small_entity_id integer NOT NULL
        REFERENCES small_entities(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    value text
);
```

and the following entries in your toml file

```toml
[[table]]
    name = "small_entities"

[[table]]
    name = "attachments"
```

`pggen` would generate the following structs for the two different tables.

```golang
type SmallEntity struct {
	Id                       int64                         `gorm:"column:id;is_primary"`
	Anint                    int64                         `gorm:"column:anint"`
	Attachments              []*Attachment                 `gorm:"foreignKey:SmallEntityId"`
}

type Attachment struct {
	Id            uuid.UUID `gorm:"column:id;is_primary"`
	SmallEntityId int64     `gorm:"column:small_entity_id"`
	Value         *string   `gorm:"column:value"`
	SmallEntity   *SmallEntity
}
```

If the database schema does not include a foreign key reference, you can get
pggen to generate the same sort of model structs by explicitly telling it about
the relationship with a toml entry like

```toml
[[table]]
    name = "small_entities"

[[table]]
    name = "attachments"
    [[table.belongs_to]]
        table = "small_entities"
        key_field = "small_entity_id"
```

There are a few things worth noting here. First, the structs are not named exactly the
same thing as the tables which they are generated from. Tables conventionally have plural
names, while golang structs conventionally have singular names, so `pggen` will convert
plural table names to singular names. This is important for interop with `gorm`, as `gorm`
imposes the same rule on table vs struct names. In fact, `pggen` and `gorm` use exactly
the [same code](https://github.com/jinzhu/inflection) to determine which names are plural
and which are singular.

The second thing to note here is that `SmallEntity` has an `Attachments` field and `Attachment` has
a `SmallEntity` field, neither of these show up in the DDL for the database tables. This is because
`pggen` has noticed the foreign key constraint on the `attachments` table and inferred that `Attachment`
is a child entity of `SmallEntity`. Child entities are not automatically filled in by the various accessor
methods when a record is fetched from the database, but `pggen` does generate utility code
which can be invoked for populating them. Child entities are only attached to a generated
struct if the table which holds the foreign key is also registered in the toml file.

Lastly, struct fields are generated as either boxed or unboxed types depending on the
nullability of the corresponding columns in the DDL.

#### Generated Methods & Values

Below is a list of the methods on `PGClient` which are generated for each table registered
in the configuration file

- Methods
    - Get<Entity>
        - Given the primary key of an entity, Get<Entity> fetches the entity with that key.
    - List<Entity>
        - Given a list of primary keys, List<Entity> returns a unordered list of entities
          with the given primary keys. List<Entity> always returns either exactly as many
          entities as were requested or an error (i.e. partial successes are treated as failures).
    - Insert<Entity>
        - Given an entity struct, Insert<Entity> inserts it into the database and returns
          the primary key of the inserted struct, or an error if the insert operation failed.
    - BulkInsert<Entity>
        - Given a list of entity structs, BulkInsert<Entity> inserts them all and returns
          the primary keys of the inserted structs. Note that it is possible for only a subset
          of the rows to be inserted if inserting some rows would violate existing database constraints.
          If the insert needs to be fully atomic, you can wrap the call to BulkInsert in a transaction.
    - Update<Entity>
        - Given an entity struct and a bitset, Update<Entity> updates all the fields of the
          given struct with their corresponding bit set in the database and returns the
          primary key of the updated record.
    - Upsert<Entity>
        - Given an entity, a list of conflict targets, and a bitset, Upsert<Entity> tries
          to insert the given entity. A nil list of conflict targets will default to the primary
          key for the table. If the bit for the primary key is set in the bitset
          it will try to insert the primary key from the provided entity, otherwise it will
          let the database supply a new primary key. In the event of a conflict on any of the
          provided conflict targets, Upsert<Entity> will update only those fields which are
          specified by the given bitset.
    - BulkUpsert<Entity>
        - BulkUpsert<Entity> behaves exactly like Upsert<Entity> except that it operates on
          whole a set of entities at once.
    - Delete<Entity>
        - Given the id of an entity, Delete<Entity> deletes it and returns an error on failure or
          nil on success.
    - BulkDelete<Entity>
        - Given a list of entity ids, BulkDelete<Entity> deletes all of the entities
          and returns an error on failure or nil on success.
    - <Entity>FillIncludes
        - Given a pointer to an entity and an include spec, <Entity>FillIncludes fills
          in all the decendant entities in the spec recursivly. This api allows finer grained
          control over which decendant entities are loaded from the database. For more infomation
          about include specs see [the README for that package](include/README.md).
          For entities without children, this routine is a no-op. It returns an error on failure and
          nil on success. It returns an error on failure and nil on success.
    - <Entity>BulkFillIncludes
        - Given a list of pointers to entities and an include spec, <Entity>BulkFillIncludes fills
          in all the decendant entities in the spec recursivly. It returns an error on failure and
          nil on success.
- Values (constant or variable definitions)
    - <Entity><FieldName>FieldIndex
        - For each field in the entity, `pggen` generates a constant indicating the field's
          index (0-based). These constants are useful when working with the bitset that gets
          passed to Update<Entity>.
    - <Entity>MaxFieldIndex
        - The largest valid field index for the given entity.
    - <Entity>AllFields
        - A bitset with the bits for all the fields in <Entity> set
    - <Entity>AllIncludes
        - An include spec specifying all decendant tables for use with the <Entity>FillIncludes
          method.

#### Special Fields

Mostly, `pggen` doesn't know anything about the semantics of the fields of the tables
it generates code for, it just generates type safe converters for the fields and leaves
the semantics up to higher level code. There are a few types of fields that `pggen` will
manipulate or rely on though.

##### Primary Keys

Every table that `pggen` generates a model for must have exactly one primary key field.
This is needed in order to generate all the appropriate CRUD methods, as well as for
resolving relationships between tables.

##### Foreign Keys

`pggen` will infer relationships between tables based on the foreign key constraints
established between different tables in postgres.

##### Timestamps

It is very common for database objects to have some timestamps associated with them for
tracking the life cycle of the object. By default, `pggen` won't do anything with
timestamps, but if the `created_at_field`, `updated_at_field`, or `deleted_at_field`
keys are set, either globally or on a specific table in the toml file, `pggen`
will generate `Update` and `Insert` methods that automatically keep the
corresponding timestamp fields up to date.

#### Relationships Between Tables

In addition to generating code to make working with the fields of a single struct easy,
`pggen` can automatically detect relationships between tables via foreign key constraints
in the database. If `pggen` notices a foreign key from table A to table B, it will assume
that A belongs to B, and generate a member field in the generated struct for table B
containing a slice of As. If there is a `UNIQUE` index on the foreign key, `pggen` will
infer a 1-1 relationship rather than a 1-many relationship and generate a pointer member
rather than a slice member. In the event that these defaults do not match up perfectly
with your data model `pggen` provides configuration options to explicitly control the
creation of 1-1 and 1-many relationships.

### Statements

Sometimes you want to execute SQL commands for side effects rather than for a set of
query results. To support these use cases `pggen` supports registering statements in
the config file. Shims generated for statements return `(sql.Result, error)` rather than
a slice of query results or an error. For example, to perform a custom insert you might write
the following in your config file

```toml
[[statement]]
    name = "MyInsertSmallEntity"
    body = '''
    INSERT INTO small_entities (anint) VALUES ($1)
    '''
```

which would generate a shim with the signature

```
MyInsertSmallEntity(ctx context.Context, arg0 int64) (sql.Result, error)
```

### GORM Compatibility

`pggen` aims to generate models which are compatible with the `gorm` tool. We have a lot
of code which uses `gorm` already and some people may prefer using `gorm` over the
routines that `pggen` provides. `pggen` can still help those people by taking care of
the drudge work of writing model structs which match up with the database table
definitions. pggen's GORM compatibility is focused on covering the most common
cases that we've encountered in practice, so it may not be complete. If you encounter
a way in which pggen-generated structs are not compatible with GORM usage, try
using the `field_tags` configuration option on the table block to inject custom
annotations into the generated code. Additionally, please report the incompatibility
in the issue tracker.

# Stability

`pggen` follows semver. Any breaking change will be indicated by an appropriate bump
of the version number as defined by the semver spec.

The minimum supported go language version of pggen is 1.11. `pggen` will not consider
a msgv (minimum supported go version) bump breaking for the purposes of semver, but it
will only bump the msgv for a good reason (such as that language version reaching end
of life or a very significant language feature). The msgv version will never accidentally
change, and if a previously supported go version breaks without some indication that it was
intentional, you should file a bug report.

# Comparison with Similar Tools

`pggen` is not the only database first code generator for go that works with postgres out
there.

## pggen vs [xo](https://github.com/xo/xo)

`pggen` and `xo` are similar in many ways. Both are command line tools which connect directly
to the database to read its schema and generate code to wrap both tables and queries.
Beyond the high level similarities there are a number of differences in features and
design philosophy outlined in the table below.

## pggen vs [sqlc](https://github.com/kyleconroy/sqlc)

`sqlc` is fairly unique among database first code generators in that it understands a
database schema by parsing DDL statements rather than by connecting to the database and
interrogating it. This allows sqlc to offer a simpler build system integration story than
any other database first code generator we are aware of, including pggen. The tradeoff
for implementing schema parsing this way is that sqlc needs to maintain a mirror
implementation of the type inference algorithm of each of the RDBMSs that it supports,
with all the compatibility bugs that implies.

## Database First Code Generator Feature Comparison Matrix

|            | `pggen` | `xo` | `sqlc` | notes |
|------------|---------|------|--------|-------|
| Multiple RDBMS support | no | yes | yes | We trapped ourselves with the name here a bit, but if we did want to support another RDBMS we would be able to re-use much of the `pggen` internals in new binaries. |
| Configuration | toml file | command line flags | magic comments in sql files | File based configuration like `sqlc` and `pggen` use can mean faster code generation. |
| Supports custom code generation templates | no | yes | no | |
| Tries to generate idiomatic code | no (prefers correct code) | yes | yes | A big difference here is the way that the different libraries scan results sets. Both `xo` and `sqlc` generate straight line “obvious” code that is easy to read, but `pggen` generates more complex code which is harder to understand but will not error unexpectedly when a database migration is run while the application is running. |
| Infers relationships between tables | yes | no | no | `pggen` will automatically add pointer and slice fields connecting tables associated with one another via foreign keys. These tables can be filled in with `pggen`'s includes system. |
| Transaction/Connection Support | Yes (explicit calls on the generated client) | Yes (via a wrapper interface for the database handle) | Yes (via a wrapper interface for the database handle) | Both `xo` and `sqlc` accept interfaces containing a subset of the methods on `*sql.DB` which allows the same code to be used for connection pools, connections, and transactions. `pggen` provides a specific wrapper struct for each of the different database connection handle types. |
| Style of Generated Query Routine | methods | free functions | methods | `xo` generates free functions which accept the database connection interface as well as the parameters, while `pggen` and `sqlc` attach queries to a single handle struct that contains all operations |
| Lifecycle timestamp support | yes | no | no | `pggen` can be configured to automatically fill in `created_at` and `deleted_at` timestamps. |
| Soft deletion support | yes | no | no | `pggen` can be configured to respect and fill in life cycle timestamps with its default CRUD methods, though custom queries still need to manually account for these soft deletion timestamps.|
| Can be called as a library in addition to a CLI | yes | no | no | Opendoor internally uses the library interface to better integrate with a bazel based build system. |
| Volume of generated code | high | medium | medium | If a small generated database access layer is important to you, `pggen` may not yet be the best choice for you |
| Default update allows partial updates | yes | no | n/a | `pggen`'s update CRUD routines allow you to configure which fields are updated with a bitset. Both `sqlc` and `xo` can use custom statements to handle granular updates, but they cannot deal with dynamically choosing which fields to update at runtime quite as easily. |
| Default upsert support | yes | yes | no | `pggen`'s upsert is more flexible but not as simple to use, while `xo`'s upsert is a little simpler to work with. You must use a custom statement for upsert with `sqlc`. |
| Infers good names for query arguments | no | no | yes | `sqlc` can automatically infer names for query arguments in the generated go code by noticing which fields the arguments are compared with. This type of feature is possible due to `sqlc`'s unique approach to getting database schema metadata. With `pggen`, you must explicitly configure names if you want them to be better than arg0. |
| Supports RETURNING | no | no | yes | Due to the way that `pggen` and `xo` get database metadata by creating temporary views, they are unable to support the RETURNING keyword in queries. Because `sqlc` parses the database schema, it can more easily support RETURNING. |
| Representation of NULL values | pointers | `Null*` types from the `"database/sql"` package | `Null*` types from the `"database/sql"` package | Here `pggen` chooses to expose nullable values as boxed values, which is less efficient than using the `Null*` types from the `"database/sql"`, but we believe is more ergonomic. |
| Generates code for all tables in schema | no | yes | yes | `pggen` only generates code for tables that you have explicitly asked it to generate code for. |

