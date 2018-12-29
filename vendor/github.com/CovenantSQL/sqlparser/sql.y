/*
Copyright 2017 Google Inc.
Copyright 2018 The CovenantSQL Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

%{
package sqlparser

func setParseTree(yylex interface{}, stmt Statement) {
  yylex.(*Tokenizer).ParseTree = stmt
}

func setAllowComments(yylex interface{}, allow bool) {
  yylex.(*Tokenizer).AllowComments = allow
}

func setDDL(yylex interface{}, ddl *DDL) {
  yylex.(*Tokenizer).partialDDL = ddl
}

func incNesting(yylex interface{}) bool {
  yylex.(*Tokenizer).nesting++
  if yylex.(*Tokenizer).nesting == 200 {
    return true
  }
  return false
}

func decNesting(yylex interface{}) {
  yylex.(*Tokenizer).nesting--
}

// forceEOF forces the lexer to end prematurely. Not all SQL statements
// are supported by the Parser, thus calling forceEOF will make the lexer
// return EOF early.
func forceEOF(yylex interface{}) {
  yylex.(*Tokenizer).ForceEOF = true
}

%}

%union {
  empty         struct{}
  statement     Statement
  selStmt       SelectStatement
  ddl           *DDL
  ins           *Insert
  byt           byte
  bytes         []byte
  bytes2        [][]byte
  str           string
  strs          []string
  selectExprs   SelectExprs
  selectExpr    SelectExpr
  columns       Columns
  colName       *ColName
  tableExprs    TableExprs
  tableExpr     TableExpr
  joinCondition JoinCondition
  tableName     TableName
  expr          Expr
  exprs         Exprs
  boolVal       BoolVal
  colTuple      ColTuple
  values        Values
  valTuple      ValTuple
  subquery      *Subquery
  whens         []*When
  when          *When
  orderBy       OrderBy
  order         *Order
  limit         *Limit
  updateExprs   UpdateExprs
  setExprs      SetExprs
  updateExpr    *UpdateExpr
  setExpr       *SetExpr
  colIdent      ColIdent
  tableIdent    TableIdent
  convertType   *ConvertType
  aliasedTableName *AliasedTableExpr
  TableSpec  *TableSpec
  columnType    ColumnType
  colKeyOpt     ColumnKeyOption
  optVal        *SQLVal
  LengthScaleOption LengthScaleOption
  columnDefinition *ColumnDefinition
  indexDefinition *IndexDefinition
  indexInfo     *IndexInfo
  indexColumn   *IndexColumn
  indexColumns  []*IndexColumn
}

%token LEX_ERROR
%left <bytes> UNION
%token <bytes> SELECT INSERT UPDATE DELETE FROM WHERE GROUP HAVING ORDER BY LIMIT OFFSET
%token <bytes> ALL DISTINCT AS EXISTS ASC DESC INTO KEY DEFAULT SET
%token <bytes> VALUES LAST_INSERT_ID
%left <bytes> JOIN LEFT RIGHT INNER OUTER CROSS NATURAL
%left <bytes> ON USING
%token <empty> '(' ',' ')'
%token <bytes> ID HEX STRING INTEGRAL FLOAT HEXNUM VALUE_ARG POS_ARG LIST_ARG COMMENT
%token <bytes> NULL TRUE FALSE
%token <bytes> FULL COLUMNS

// Precedence dictated by mysql. But the vitess grammar is simplified.
// Some of these operators don't conflict in our situation. Nevertheless,
// it's better to have these listed in the correct order. Also, we don't
// support all operators yet.
%left <bytes> OR
%left <bytes> AND
%right <bytes> NOT '!'
%left <bytes> BETWEEN CASE WHEN THEN ELSE END
%left <bytes> '=' '<' '>' LE GE NE IS LIKE REGEXP IN NULL_SAFE_NOTEQUAL
%left <bytes> '|'
%left <bytes> '&'
%left <bytes> SHIFT_LEFT SHIFT_RIGHT
%left <bytes> '+' '-'
%left <bytes> '*' '/' DIV '%' MOD
%left <bytes> '^'
%right <bytes> '~' UNARY
%right <bytes> INTERVAL
%nonassoc <bytes> '.'

// DDL Tokens
%token <bytes> CREATE ALTER DROP RENAME ADD
%token <bytes> TABLE INDEX TO IGNORE IF UNIQUE PRIMARY COLUMN CONSTRAINT FOREIGN
%token <bytes> SHOW DESCRIBE DATE ESCAPE EXPLAIN

// Type Tokens
%token <bytes> TINYINT SMALLINT MEDIUMINT INT INTEGER BIGINT INTNUM
%token <bytes> REAL DOUBLE FLOAT_TYPE DECIMAL NUMERIC
%token <bytes> TIME TIMESTAMP DATETIME YEAR
%token <bytes> CHAR VARCHAR BOOL NCHAR
%token <bytes> TEXT TINYTEXT MEDIUMTEXT LONGTEXT
%token <bytes> BLOB TINYBLOB MEDIUMBLOB LONGBLOB

// Type Modifiers
%token <bytes> AUTO_INCREMENT SIGNED UNSIGNED ZEROFILL

// Supported SHOW tokens
%token <bytes> TABLES

// Functions
%token <bytes> CURRENT_TIMESTAMP CURRENT_DATE CURRENT_TIME
%token <bytes> REPLACE
%token <bytes> CAST
%token <bytes> GROUP_CONCAT SEPARATOR

// MySQL reserved words that are unused by this grammar will map to this token.
%token <bytes> UNUSED

%type <statement> command
%type <selStmt> select_statement base_select union_lhs union_rhs
%type <statement> insert_statement update_statement delete_statement
%type <statement> create_statement alter_statement drop_statement
%type <ddl> create_table_prefix
%type <statement> show_statement other_statement
%type <bytes2> comment_opt comment_list
%type <str> union_op insert_or_replace
%type <str> distinct_opt separator_opt
%type <expr> like_escape_opt
%type <selectExprs> select_expression_list select_expression_list_opt
%type <selectExpr> select_expression
%type <expr> expression
%type <tableExprs> from_opt table_references
%type <tableExpr> table_reference table_factor join_table
%type <joinCondition> join_condition join_condition_opt
%type <str> inner_join outer_join natural_join
%type <tableName> table_name into_table_name
%type <aliasedTableName> aliased_table_name
%type <expr> where_expression_opt
%type <expr> condition
%type <boolVal> boolean_value
%type <str> compare
%type <ins> insert_data
%type <expr> value value_expression
%type <expr> function_call_keyword function_call_nonkeyword function_call_generic function_call_conflict
%type <str> is_suffix
%type <colTuple> col_tuple
%type <exprs> expression_list
%type <values> tuple_list
%type <valTuple> row_tuple tuple_or_empty
%type <expr> tuple_expression
%type <subquery> subquery
%type <colName> column_name
%type <whens> when_expression_list
%type <when> when_expression
%type <expr> expression_opt else_expression_opt
%type <exprs> group_by_opt
%type <expr> having_opt
%type <orderBy> order_by_opt order_list
%type <order> order
%type <str> asc_desc_opt
%type <limit> limit_opt
%type <columns> ins_column_list column_list
%type <updateExprs> update_list
%type <updateExpr> update_expression
%type <str> ignore_opt
%type <byt> exists_opt
%type <empty> not_exists_opt constraint_opt
%type <bytes> reserved_keyword non_reserved_keyword
%type <colIdent> sql_id reserved_sql_id col_alias as_ci_opt
%type <tableIdent> table_id reserved_table_id table_alias as_opt_id
%type <empty> as_opt
%type <empty> ddl_force_eof
%type <convertType> convert_type
%type <columnType> column_type
%type <columnType> int_type decimal_type numeric_type time_type char_type
%type <optVal> length_opt column_default_opt
%type <boolVal> unsigned_opt zero_fill_opt
%type <LengthScaleOption> float_length_opt decimal_length_opt
%type <boolVal> null_opt auto_increment_opt
%type <colKeyOpt> column_key_opt
%type <columnDefinition> column_definition
%type <indexDefinition> index_definition
%type <str> index_or_key
%type <TableSpec> table_spec table_column_list
%type <str> table_option_list table_option table_opt_value
%type <indexInfo> index_info
%type <indexColumn> index_column
%type <indexColumns> index_column_list
%type <bytes> alter_object_type
%type <bytes> full_opt

%start any_command

%%

any_command:
  command semicolon_opt
  {
    setParseTree(yylex, $1)
  }

semicolon_opt:
/*empty*/ {}
| ';' {}

command:
  select_statement
  {
    $$ = $1
  }
| insert_statement
| update_statement
| delete_statement
| create_statement
| alter_statement
| drop_statement
| show_statement
| other_statement

select_statement:
  base_select order_by_opt limit_opt
  {
    sel := $1.(*Select)
    sel.OrderBy = $2
    sel.Limit = $3
    $$ = sel
  }
| union_lhs union_op union_rhs order_by_opt limit_opt
  {
    $$ = &Union{Type: $2, Left: $1, Right: $3, OrderBy: $4, Limit: $5}
  }

// base_select is an unparenthesized SELECT with no order by clause or beyond.
base_select:
  SELECT comment_opt distinct_opt select_expression_list from_opt where_expression_opt group_by_opt having_opt
  {
    $$ = &Select{Comments: Comments($2), Distinct: $3, SelectExprs: $4, From: $5, Where: NewWhere(WhereStr, $6), GroupBy: GroupBy($7), Having: NewWhere(HavingStr, $8)}
  }
| SELECT comment_opt distinct_opt select_expression_list
  {
    $$ = &Select{Comments: Comments($2), Distinct: $3, SelectExprs: $4}
  }

union_lhs:
  select_statement
  {
    $$ = $1
  }
| openb select_statement closeb
  {
    $$ = &ParenSelect{Select: $2}
  }

union_rhs:
  base_select
  {
    $$ = $1
  }
| openb select_statement closeb
  {
    $$ = &ParenSelect{Select: $2}
  }


insert_statement:
  insert_or_replace comment_opt ignore_opt into_table_name insert_data
  {
    // insert_data returns a *Insert pre-filled with Columns & Values
    ins := $5
    ins.Action = $1
    ins.Comments = $2
    ins.Ignore = $3
    ins.Table = $4
    $$ = ins
  }
| insert_or_replace comment_opt ignore_opt into_table_name SET update_list
  {
    cols := make(Columns, 0, len($6))
    vals := make(ValTuple, 0, len($6))
    for _, updateList := range $6 {
      cols = append(cols, updateList.Name.Name)
      vals = append(vals, updateList.Expr)
    }
    $$ = &Insert{Action: $1, Comments: Comments($2), Ignore: $3, Table: $4, Columns: cols, Rows: Values{vals}}
  }

insert_or_replace:
  INSERT
  {
    $$ = InsertStr
  }
| REPLACE
  {
    $$ = ReplaceStr
  }
| INSERT OR REPLACE
  {
    $$ = ReplaceStr
  }

update_statement:
  UPDATE comment_opt table_references SET update_list where_expression_opt order_by_opt limit_opt
  {
    $$ = &Update{Comments: Comments($2), TableExprs: $3, Exprs: $5, Where: NewWhere(WhereStr, $6), OrderBy: $7, Limit: $8}
  }

delete_statement:
  DELETE comment_opt FROM table_name where_expression_opt order_by_opt limit_opt
  {
    $$ = &Delete{Comments: Comments($2), TableExprs:  TableExprs{&AliasedTableExpr{Expr:$4}}, Where: NewWhere(WhereStr, $5), OrderBy: $6, Limit: $7}
  }

create_statement:
  create_table_prefix table_spec
  {
    $1.TableSpec = $2
    $$ = $1
  }
| CREATE constraint_opt INDEX not_exists_opt ID ON table_name ddl_force_eof
  {
    // Change this to an alter statement
    $$ = &DDL{Action: CreateIndexStr, Table: $7, NewName:$7}
  }

create_table_prefix:
  CREATE TABLE not_exists_opt table_name
  {
    $$ = &DDL{Action: CreateStr, NewName: $4}
    setDDL(yylex, $$)
  }

table_spec:
  '(' table_column_list ')' table_option_list
  {
    $$ = $2
    $$.Options = $4
  }

table_column_list:
  column_definition
  {
    $$ = &TableSpec{}
    $$.AddColumn($1)
  }
| table_column_list ',' column_definition
  {
    $$.AddColumn($3)
  }
| table_column_list ',' index_definition
  {
    $$.AddIndex($3)
  }

column_definition:
  ID column_type null_opt column_default_opt auto_increment_opt column_key_opt
  {
    $2.NotNull = $3
    $2.Default = $4
    $2.Autoincrement = $5
    $2.KeyOpt = $6
    $$ = &ColumnDefinition{Name: NewColIdent(string($1)), Type: $2}
  }
column_type:
  numeric_type unsigned_opt zero_fill_opt
  {
    $$ = $1
    $$.Unsigned = $2
    $$.Zerofill = $3
  }
| char_type
| time_type

numeric_type:
  int_type length_opt
  {
    $$ = $1
    $$.Length = $2
  }
| decimal_type
  {
    $$ = $1
  }

int_type:
  TINYINT
  {
    $$ = ColumnType{Type: string($1)}
  }
| SMALLINT
  {
    $$ = ColumnType{Type: string($1)}
  }
| MEDIUMINT
  {
    $$ = ColumnType{Type: string($1)}
  }
| INT
  {
    $$ = ColumnType{Type: string($1)}
  }
| INTEGER
  {
    $$ = ColumnType{Type: string($1)}
  }
| BIGINT
  {
    $$ = ColumnType{Type: string($1)}
  }

decimal_type:
REAL float_length_opt
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| DOUBLE float_length_opt
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| FLOAT_TYPE float_length_opt
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| DECIMAL decimal_length_opt
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| NUMERIC decimal_length_opt
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }

time_type:
  DATE
  {
    $$ = ColumnType{Type: string($1)}
  }
| TIME length_opt
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| TIMESTAMP length_opt
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| DATETIME length_opt
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| YEAR
  {
    $$ = ColumnType{Type: string($1)}
  }

char_type:
  CHAR length_opt
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| VARCHAR length_opt
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| TEXT
  {
    $$ = ColumnType{Type: string($1)}
  }
| TINYTEXT
  {
    $$ = ColumnType{Type: string($1)}
  }
| MEDIUMTEXT
  {
    $$ = ColumnType{Type: string($1)}
  }
| LONGTEXT
  {
    $$ = ColumnType{Type: string($1)}
  }
| BLOB
  {
    $$ = ColumnType{Type: string($1)}
  }
| TINYBLOB
  {
    $$ = ColumnType{Type: string($1)}
  }
| MEDIUMBLOB
  {
    $$ = ColumnType{Type: string($1)}
  }
| LONGBLOB
  {
    $$ = ColumnType{Type: string($1)}
  }

length_opt:
  {
    $$ = nil
  }
| '(' INTEGRAL ')'
  {
    $$ = NewIntVal($2)
  }

float_length_opt:
  {
    $$ = LengthScaleOption{}
  }
| '(' INTEGRAL ',' INTEGRAL ')'
  {
    $$ = LengthScaleOption{
        Length: NewIntVal($2),
        Scale: NewIntVal($4),
    }
  }

decimal_length_opt:
  {
    $$ = LengthScaleOption{}
  }
| '(' INTEGRAL ')'
  {
    $$ = LengthScaleOption{
        Length: NewIntVal($2),
    }
  }
| '(' INTEGRAL ',' INTEGRAL ')'
  {
    $$ = LengthScaleOption{
        Length: NewIntVal($2),
        Scale: NewIntVal($4),
    }
  }

unsigned_opt:
  {
    $$ = BoolVal(false)
  }
| UNSIGNED
  {
    $$ = BoolVal(true)
  }

zero_fill_opt:
  {
    $$ = BoolVal(false)
  }
| ZEROFILL
  {
    $$ = BoolVal(true)
  }

// Null opt returns false to mean NULL (i.e. the default) and true for NOT NULL
null_opt:
  {
    $$ = BoolVal(false)
  }
| NULL
  {
    $$ = BoolVal(false)
  }
| NOT NULL
  {
    $$ = BoolVal(true)
  }

column_default_opt:
  {
    $$ = nil
  }
| DEFAULT STRING
  {
    $$ = NewStrVal($2)
  }
| DEFAULT INTEGRAL
  {
    $$ = NewIntVal($2)
  }
| DEFAULT FLOAT
  {
    $$ = NewFloatVal($2)
  }
| DEFAULT NULL
  {
    $$ = NewValArg($2)
  }
| DEFAULT CURRENT_TIMESTAMP
  {
    $$ = NewValArg($2)
  }

auto_increment_opt:
  {
    $$ = BoolVal(false)
  }
| AUTO_INCREMENT
  {
    $$ = BoolVal(true)
  }

column_key_opt:
  {
    $$ = colKeyNone
  }
| PRIMARY KEY
  {
    $$ = colKeyPrimary
  }
| KEY
  {
    $$ = colKey
  }
| UNIQUE KEY
  {
    $$ = colKeyUniqueKey
  }
| UNIQUE
  {
    $$ = colKeyUnique
  }

index_definition:
  index_info '(' index_column_list ')'
  {
    $$ = &IndexDefinition{Info: $1, Columns: $3}
  }

index_info:
  PRIMARY KEY
  {
    $$ = &IndexInfo{Type: string($1) + " " + string($2), Name: NewColIdent("PRIMARY"), Primary: true, Unique: true}
  }
| UNIQUE index_or_key ID
  {
    $$ = &IndexInfo{Type: string($1) + " " + string($2), Name: NewColIdent(string($3)), Unique: true}
  }
| UNIQUE ID
  {
    $$ = &IndexInfo{Type: string($1), Name: NewColIdent(string($2)), Unique: true}
  }
| index_or_key ID
  {
    $$ = &IndexInfo{Type: string($1), Name: NewColIdent(string($2)), Unique: false}
  }

index_or_key:
    INDEX
  {
    $$ = string($1)
  }
  | KEY
  {
    $$ = string($1)
  }

index_column_list:
  index_column
  {
    $$ = []*IndexColumn{$1}
  }
| index_column_list ',' index_column
  {
    $$ = append($$, $3)
  }

index_column:
  sql_id length_opt
  {
      $$ = &IndexColumn{Column: $1, Length: $2}
  }

table_option_list:
  {
    $$ = ""
  }
| table_option
  {
    $$ = " " + string($1)
  }
| table_option_list ',' table_option
  {
    $$ = string($1) + ", " + string($3)
  }

// rather than explicitly parsing the various keywords for table options,
// just accept any number of keywords, IDs, strings, numbers, and '='
table_option:
  table_opt_value
  {
    $$ = $1
  }
| table_option table_opt_value
  {
    $$ = $1 + " " + $2
  }
| table_option '=' table_opt_value
  {
    $$ = $1 + "=" + $3
  }

table_opt_value:
  reserved_sql_id
  {
    $$ = $1.String()
  }
| STRING
  {
    $$ = "'" + string($1) + "'"
  }
| INTEGRAL
  {
    $$ = string($1)
  }

alter_statement:
  ALTER TABLE table_name ADD alter_object_type column_definition
  {
    $$ = &DDL{Action: AlterStr, Table: $3, NewName: $3}
  }
| ALTER TABLE table_name RENAME TO table_name
  {
    // Change this to a rename statement
    $$ = &DDL{Action: RenameStr, Table: $3, NewName: $6}
  }
| ALTER TABLE table_name RENAME alter_object_type column_name TO column_name
  {
    // Rename an index can just be an alter
    $$ = &DDL{Action: AlterStr, Table: $3, NewName: $3}
  }

alter_object_type:
  {} | COLUMN

drop_statement:
  DROP TABLE exists_opt table_name
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }
    $$ = &DDL{Action: DropStr, Table: $4, IfExists: exists}
  }
| DROP INDEX exists_opt table_name
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }
    $$ = &DDL{Action: DropIndexStr, Table: $4, IfExists: exists}
  }

show_statement:
SHOW CREATE TABLE table_name
  {
    $$ = &Show{Type: string($3), ShowCreate: true, OnTable: $4}
  }
| SHOW INDEX FROM TABLE table_name
  {
    $$ = &Show{Type: string($2), OnTable: $5}
  }
| SHOW TABLE table_name
  {
    $$ = &Show{Type: string($2), OnTable: $3}
  }
| SHOW full_opt TABLES
  {
    $$ = &Show{Type: string($3)}
  }
| SHOW full_opt COLUMNS FROM table_name
  {
    $$ = &Show{Type: "table", OnTable: $5}
  }

full_opt:
  {
    $$ = nil
  }
| FULL
  {
    $$ = nil
  }

other_statement:
  DESC table_name
  {
    $$ = &Show{Type: "table", OnTable: $2}
  }
| DESCRIBE table_name
  {
    $$ = &Show{Type: "table", OnTable: $2}
  }
| EXPLAIN force_eof
  {
    $$ = &Explain{}
  }

comment_opt:
  {
    setAllowComments(yylex, true)
  }
  comment_list
  {
    $$ = $2
    setAllowComments(yylex, false)
  }

comment_list:
  {
    $$ = nil
  }
| comment_list COMMENT
  {
    $$ = append($1, $2)
  }

union_op:
  UNION
  {
    $$ = UnionStr
  }
| UNION ALL
  {
    $$ = UnionAllStr
  }

distinct_opt:
  {
    $$ = ""
  }
| DISTINCT
  {
    $$ = DistinctStr
  }

select_expression_list_opt:
  {
    $$ = nil
  }
| select_expression_list
  {
    $$ = $1
  }

select_expression_list:
  select_expression
  {
    $$ = SelectExprs{$1}
  }
| select_expression_list ',' select_expression
  {
    $$ = append($$, $3)
  }

select_expression:
  '*'
  {
    $$ = &StarExpr{}
  }
| expression as_ci_opt
  {
    $$ = &AliasedExpr{Expr: $1, As: $2}
  }
| table_id '.' '*'
  {
    $$ = &StarExpr{TableName: TableName{Name: $1}}
  }
| table_id '.' reserved_table_id '.' '*'
  {
    $$ = &StarExpr{TableName: TableName{Qualifier: $1, Name: $3}}
  }

as_ci_opt:
  {
    $$ = ColIdent{}
  }
| col_alias
  {
    $$ = $1
  }
| AS col_alias
  {
    $$ = $2
  }

col_alias:
  sql_id
| STRING
  {
    $$ = NewColIdent(string($1))
  }

from_opt:
  FROM table_references
  {
    $$ = $2
  }

table_references:
  table_reference
  {
    $$ = TableExprs{$1}
  }
| table_references ',' table_reference
  {
    $$ = append($$, $3)
  }

table_reference:
  table_factor
| join_table

table_factor:
  aliased_table_name
  {
    $$ = $1
  }
| subquery
  {
    $$ = &AliasedTableExpr{Expr:$1} 
  }
| subquery as_opt table_id
  {
    $$ = &AliasedTableExpr{Expr:$1, As: $3}
  }
| openb table_references closeb
  {
    $$ = &ParenTableExpr{Exprs: $2}
  }

aliased_table_name:
table_name as_opt_id
  {
    $$ = &AliasedTableExpr{Expr:$1, As: $2}
  }

column_list:
  sql_id
  {
    $$ = Columns{$1}
  }
| column_list ',' sql_id
  {
    $$ = append($$, $3)
  }

join_table:
  table_reference inner_join table_factor join_condition_opt
  {
    $$ = &JoinTableExpr{LeftExpr: $1, Join: $2, RightExpr: $3, Condition: $4}
  }
| table_reference outer_join table_reference join_condition
  {
    $$ = &JoinTableExpr{LeftExpr: $1, Join: $2, RightExpr: $3, Condition: $4}
  }
| table_reference natural_join table_factor
  {
    $$ = &JoinTableExpr{LeftExpr: $1, Join: $2, RightExpr: $3}
  }

join_condition:
  ON expression
  { $$ = JoinCondition{On: $2} }
| USING '(' column_list ')'
  { $$ = JoinCondition{Using: $3} }

join_condition_opt:
%prec JOIN
  { $$ = JoinCondition{} }
| join_condition
  { $$ = $1 }

as_opt:
  { $$ = struct{}{} }
| AS
  { $$ = struct{}{} }

as_opt_id:
  {
    $$ = NewTableIdent("")
  }
| table_alias
  {
    $$ = $1
  }
| AS table_alias
  {
    $$ = $2
  }

table_alias:
  table_id
| STRING
  {
    $$ = NewTableIdent(string($1))
  }

inner_join:
  JOIN
  {
    $$ = JoinStr
  }
| INNER JOIN
  {
    $$ = InnerJoinStr
  }
| CROSS JOIN
  {
    $$ = CrossJoinStr
  }

outer_join:
  LEFT JOIN
  {
    $$ = LeftJoinStr
  }
| LEFT OUTER JOIN
  {
    $$ = LeftJoinStr
  }

natural_join:
 NATURAL JOIN
  {
    $$ = NaturalJoinStr
  }
| NATURAL outer_join
  {
    $$ = NaturalLeftJoinStr
  }

into_table_name:
  INTO table_name
  {
    $$ = $2
  }
| table_name
  {
    $$ = $1
  }

table_name:
  table_id
  {
    $$ = TableName{Name: $1}
  }
| table_id '.' reserved_table_id
  {
    $$ = TableName{Qualifier: $1, Name: $3}
  }

where_expression_opt:
  {
    $$ = nil
  }
| WHERE expression
  {
    $$ = $2
  }

expression:
  condition
  {
    $$ = $1
  }
| expression AND expression
  {
    $$ = &AndExpr{Left: $1, Right: $3}
  }
| expression OR expression
  {
    $$ = &OrExpr{Left: $1, Right: $3}
  }
| NOT expression
  {
    $$ = &NotExpr{Expr: $2}
  }
| expression IS is_suffix
  {
    $$ = &IsExpr{Operator: $3, Expr: $1}
  }
| value_expression
  {
    $$ = $1
  }

boolean_value:
  TRUE
  {
    $$ = BoolVal(true)
  }
| FALSE
  {
    $$ = BoolVal(false)
  }

condition:
  value_expression compare value_expression
  {
    $$ = &ComparisonExpr{Left: $1, Operator: $2, Right: $3}
  }
| value_expression IN col_tuple
  {
    $$ = &ComparisonExpr{Left: $1, Operator: InStr, Right: $3}
  }
| value_expression NOT IN col_tuple
  {
    $$ = &ComparisonExpr{Left: $1, Operator: NotInStr, Right: $4}
  }
| value_expression LIKE value_expression like_escape_opt
  {
    $$ = &ComparisonExpr{Left: $1, Operator: LikeStr, Right: $3, Escape: $4}
  }
| value_expression NOT LIKE value_expression like_escape_opt
  {
    $$ = &ComparisonExpr{Left: $1, Operator: NotLikeStr, Right: $4, Escape: $5}
  }
| value_expression REGEXP value_expression
  {
    $$ = &ComparisonExpr{Left: $1, Operator: RegexpStr, Right: $3}
  }
| value_expression NOT REGEXP value_expression
  {
    $$ = &ComparisonExpr{Left: $1, Operator: NotRegexpStr, Right: $4}
  }
| value_expression BETWEEN value_expression AND value_expression
  {
    $$ = &RangeCond{Left: $1, Operator: BetweenStr, From: $3, To: $5}
  }
| value_expression NOT BETWEEN value_expression AND value_expression
  {
    $$ = &RangeCond{Left: $1, Operator: NotBetweenStr, From: $4, To: $6}
  }
| EXISTS subquery
  {
    $$ = &ExistsExpr{Subquery: $2}
  }

is_suffix:
  NULL
  {
    $$ = IsNullStr
  }
| NOT NULL
  {
    $$ = IsNotNullStr
  }
| TRUE
  {
    $$ = IsTrueStr
  }
| NOT TRUE
  {
    $$ = IsNotTrueStr
  }
| FALSE
  {
    $$ = IsFalseStr
  }
| NOT FALSE
  {
    $$ = IsNotFalseStr
  }

compare:
  '='
  {
    $$ = EqualStr
  }
| '<'
  {
    $$ = LessThanStr
  }
| '>'
  {
    $$ = GreaterThanStr
  }
| LE
  {
    $$ = LessEqualStr
  }
| GE
  {
    $$ = GreaterEqualStr
  }
| NE
  {
    $$ = NotEqualStr
  }
| NULL_SAFE_NOTEQUAL
 {
    $$ = NullSafeNotEqualStr
 }

like_escape_opt:
  {
    $$ = nil
  }
| ESCAPE value_expression
  {
    $$ = $2
  }

col_tuple:
  row_tuple
  {
    $$ = $1
  }
| subquery
  {
    $$ = $1
  }
| LIST_ARG
  {
    $$ = ListArg($1)
  }

subquery:
  openb select_statement closeb
  {
    $$ = &Subquery{$2}
  }

expression_list:
  expression
  {
    $$ = Exprs{$1}
  }
| expression_list ',' expression
  {
    $$ = append($1, $3)
  }

value_expression:
  value
  {
    $$ = $1
  }
| boolean_value
  {
    $$ = $1
  }
| column_name
  {
    $$ = $1
  }
| tuple_expression
  {
    $$ = $1
  }
| subquery
  {
    $$ = $1
  }
| value_expression '&' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: BitAndStr, Right: $3}
  }
| value_expression '|' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: BitOrStr, Right: $3}
  }
| value_expression '^' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: BitXorStr, Right: $3}
  }
| value_expression '+' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: PlusStr, Right: $3}
  }
| value_expression '-' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: MinusStr, Right: $3}
  }
| value_expression '*' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: MultStr, Right: $3}
  }
| value_expression '/' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: DivStr, Right: $3}
  }
| value_expression DIV value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: IntDivStr, Right: $3}
  }
| value_expression '%' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: ModStr, Right: $3}
  }
| value_expression MOD value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: ModStr, Right: $3}
  }
| value_expression SHIFT_LEFT value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: ShiftLeftStr, Right: $3}
  }
| value_expression SHIFT_RIGHT value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: ShiftRightStr, Right: $3}
  }
| '+'  value_expression %prec UNARY
  {
    if num, ok := $2.(*SQLVal); ok && num.Type == IntVal {
      $$ = num
    } else {
      $$ = &UnaryExpr{Operator: UPlusStr, Expr: $2}
    }
  }
| '-'  value_expression %prec UNARY
  {
    if num, ok := $2.(*SQLVal); ok && num.Type == IntVal {
      // Handle double negative
      if num.Val[0] == '-' {
        num.Val = num.Val[1:]
        $$ = num
      } else {
        $$ = NewIntVal(append([]byte("-"), num.Val...))
      }
    } else {
      $$ = &UnaryExpr{Operator: UMinusStr, Expr: $2}
    }
  }
| '~'  value_expression
  {
    $$ = &UnaryExpr{Operator: TildaStr, Expr: $2}
  }
| '!' value_expression %prec UNARY
  {
    $$ = &UnaryExpr{Operator: BangStr, Expr: $2}
  }
| INTERVAL value_expression sql_id
  {
    // This rule prevents the usage of INTERVAL
    // as a function. If support is needed for that,
    // we'll need to revisit this. The solution
    // will be non-trivial because of grammar conflicts.
    $$ = &IntervalExpr{Expr: $2, Unit: $3.String()}
  }
| function_call_generic
| function_call_keyword
| function_call_nonkeyword
| function_call_conflict

/*
  Regular function calls without special token or syntax, guaranteed to not
  introduce side effects due to being a simple identifier
*/
function_call_generic:
  sql_id openb select_expression_list_opt closeb
  {
    $$ = &FuncExpr{Name: $1, Exprs: $3}
  }
| sql_id openb DISTINCT select_expression_list closeb
  {
    $$ = &FuncExpr{Name: $1, Distinct: true, Exprs: $4}
  }
| table_id '.' reserved_sql_id openb select_expression_list_opt closeb
  {
    $$ = &FuncExpr{Qualifier: $1, Name: $3, Exprs: $5}
  }

/*
  Function calls using reserved keywords, with dedicated grammar rules
  as a result
*/
function_call_keyword:
  CAST openb expression AS convert_type closeb
  {
    $$ = &ConvertExpr{Expr: $3, Type: $5}
  }
| GROUP_CONCAT openb distinct_opt select_expression_list order_by_opt separator_opt closeb
  {
    $$ = &GroupConcatExpr{Distinct: $3, Exprs: $4, OrderBy: $5, Separator: $6}
  }
| CASE expression_opt when_expression_list else_expression_opt END
  {
    $$ = &CaseExpr{Expr: $2, Whens: $3, Else: $4}
  }
| VALUES openb column_name closeb
  {
    $$ = &ValuesFuncExpr{Name: $3}
  }

/*
  Function calls using non reserved keywords but with special syntax forms.
  Dedicated grammar rules are needed because of the special syntax
*/
function_call_nonkeyword:
  CURRENT_TIMESTAMP
  {
    $$ = &TimeExpr{Expr: NewColIdent("current_timestamp")}
  }
  // curdate
| CURRENT_DATE
  {
    $$ = &TimeExpr{Expr: NewColIdent("current_date")}
  }
  // curtime
| CURRENT_TIME
  {
    $$ = &TimeExpr{Expr: NewColIdent("current_time")}
  }

/*
  Function calls using non reserved keywords with *normal* syntax forms. Because
  the names are non-reserved, they need a dedicated rule so as not to conflict
*/
function_call_conflict:
  IF openb select_expression_list closeb
  {
    $$ = &FuncExpr{Name: NewColIdent("if"), Exprs: $3}
  }
| MOD openb select_expression_list closeb
  {
    $$ = &FuncExpr{Name: NewColIdent("mod"), Exprs: $3}
  }
| REPLACE openb select_expression_list closeb
  {
    $$ = &FuncExpr{Name: NewColIdent("replace"), Exprs: $3}
  }

convert_type:
  CHAR length_opt
  {
    $$ = &ConvertType{Type: string($1), Length: $2}
  }
| DATE
  {
    $$ = &ConvertType{Type: string($1)}
  }
| DATETIME length_opt
  {
    $$ = &ConvertType{Type: string($1), Length: $2}
  }
| DECIMAL decimal_length_opt
  {
    $$ = &ConvertType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| NCHAR length_opt
  {
    $$ = &ConvertType{Type: string($1), Length: $2}
  }
| SIGNED
  {
    $$ = &ConvertType{Type: string($1)}
  }
| SIGNED INTEGER
  {
    $$ = &ConvertType{Type: string($1)}
  }
| TIME length_opt
  {
    $$ = &ConvertType{Type: string($1), Length: $2}
  }
| UNSIGNED
  {
    $$ = &ConvertType{Type: string($1)}
  }
| UNSIGNED INTEGER
  {
    $$ = &ConvertType{Type: string($1)}
  }
| TIMESTAMP length_opt
 {
    $$ = &ConvertType{Type: string($1), Length: $2}
 }
| INT length_opt
 {
    $$ = &ConvertType{Type: string($1), Length: $2}
 }
| TINYINT length_opt
 {
    $$ = &ConvertType{Type: string($1), Length: $2}
 }
| SMALLINT length_opt
 {
    $$ = &ConvertType{Type: string($1), Length: $2}
 }
| MEDIUMINT length_opt
 {
    $$ = &ConvertType{Type: string($1), Length: $2}
 }
| BIGINT length_opt
 {
    $$ = &ConvertType{Type: string($1), Length: $2}
 }
| STRING
 {
    $$ = &ConvertType{Type: string($1)}
 }

expression_opt:
  {
    $$ = nil
  }
| expression
  {
    $$ = $1
  }

separator_opt:
  {
    $$ = string("")
  }
| SEPARATOR STRING
  {
    $$ = " separator '"+string($2)+"'"
  }

when_expression_list:
  when_expression
  {
    $$ = []*When{$1}
  }
| when_expression_list when_expression
  {
    $$ = append($1, $2)
  }

when_expression:
  WHEN expression THEN expression
  {
    $$ = &When{Cond: $2, Val: $4}
  }

else_expression_opt:
  {
    $$ = nil
  }
| ELSE expression
  {
    $$ = $2
  }

column_name:
  sql_id
  {
    $$ = &ColName{Name: $1}
  }
| table_id '.' reserved_sql_id
  {
    $$ = &ColName{Qualifier: TableName{Name: $1}, Name: $3}
  }
| table_id '.' reserved_table_id '.' reserved_sql_id
  {
    $$ = &ColName{Qualifier: TableName{Qualifier: $1, Name: $3}, Name: $5}
  }

value:
  STRING
  {
    $$ = NewStrVal($1)
  }
| HEX
  {
    $$ = NewHexVal($1)
  }
| INTEGRAL
  {
    $$ = NewIntVal($1)
  }
| FLOAT
  {
    $$ = NewFloatVal($1)
  }
| HEXNUM
  {
    $$ = NewHexNum($1)
  }
| VALUE_ARG
  {
    $$ = NewValArg($1)
  }
| POS_ARG
  {
    $$ = NewPosArg($1)
  }
| NULL
  {
    $$ = &NullVal{}
  }

group_by_opt:
  {
    $$ = nil
  }
| GROUP BY expression_list
  {
    $$ = $3
  }

having_opt:
  {
    $$ = nil
  }
| HAVING expression
  {
    $$ = $2
  }

order_by_opt:
  {
    $$ = nil
  }
| ORDER BY order_list
  {
    $$ = $3
  }

order_list:
  order
  {
    $$ = OrderBy{$1}
  }
| order_list ',' order
  {
    $$ = append($1, $3)
  }

order:
  expression asc_desc_opt
  {
    $$ = &Order{Expr: $1, Direction: $2}
  }

asc_desc_opt:
  {
    $$ = AscScr
  }
| ASC
  {
    $$ = AscScr
  }
| DESC
  {
    $$ = DescScr
  }

limit_opt:
  {
    $$ = nil
  }
| LIMIT expression
  {
    $$ = &Limit{Rowcount: $2}
  }
| LIMIT expression ',' expression
  {
    $$ = &Limit{Offset: $2, Rowcount: $4}
  }
| LIMIT expression OFFSET expression
  {
    $$ = &Limit{Offset: $4, Rowcount: $2}
  }

// insert_data expands all combinations into a single rule.
// This avoids a shift/reduce conflict while encountering the
// following two possible constructs:
// insert into t1(a, b) (select * from t2)
// insert into t1(select * from t2)
// Because the rules are together, the parser can keep shifting
// the tokens until it disambiguates a as sql_id and select as keyword.
insert_data:
  VALUES tuple_list
  {
    $$ = &Insert{Rows: $2}
  }
| select_statement
  {
    $$ = &Insert{Rows: $1}
  }
| openb select_statement closeb
  {
    // Drop the redundant parenthesis.
    $$ = &Insert{Rows: $2}
  }
| openb ins_column_list closeb VALUES tuple_list
  {
    $$ = &Insert{Columns: $2, Rows: $5}
  }
| openb ins_column_list closeb select_statement
  {
    $$ = &Insert{Columns: $2, Rows: $4}
  }
| openb ins_column_list closeb openb select_statement closeb
  {
    // Drop the redundant parenthesis.
    $$ = &Insert{Columns: $2, Rows: $5}
  }

ins_column_list:
  sql_id
  {
    $$ = Columns{$1}
  }
| sql_id '.' sql_id
  {
    $$ = Columns{$3}
  }
| ins_column_list ',' sql_id
  {
    $$ = append($$, $3)
  }
| ins_column_list ',' sql_id '.' sql_id
  {
    $$ = append($$, $5)
  }

tuple_list:
  tuple_or_empty
  {
    $$ = Values{$1}
  }
| tuple_list ',' tuple_or_empty
  {
    $$ = append($1, $3)
  }

tuple_or_empty:
  row_tuple
  {
    $$ = $1
  }
| openb closeb
  {
    $$ = ValTuple{}
  }

row_tuple:
  openb expression_list closeb
  {
    $$ = ValTuple($2)
  }

tuple_expression:
  row_tuple
  {
    if len($1) == 1 {
      $$ = &ParenExpr{$1[0]}
    } else {
      $$ = $1
    }
  }

update_list:
  update_expression
  {
    $$ = UpdateExprs{$1}
  }
| update_list ',' update_expression
  {
    $$ = append($1, $3)
  }

update_expression:
  column_name '=' expression
  {
    $$ = &UpdateExpr{Name: $1, Expr: $3}
  }

exists_opt:
  { $$ = 0 }
| IF EXISTS
  { $$ = 1 }

not_exists_opt:
  { $$ = struct{}{} }
| IF NOT EXISTS
  { $$ = struct{}{} }

ignore_opt:
  { $$ = "" }
| IGNORE
  { $$ = IgnoreStr }

constraint_opt:
  { $$ = struct{}{} }
| UNIQUE
  { $$ = struct{}{} }
| sql_id
  { $$ = struct{}{} }

sql_id:
  ID
  {
    $$ = NewColIdent(string($1))
  }
| non_reserved_keyword
  {
    $$ = NewColIdent(string($1))
  }

reserved_sql_id:
  sql_id
| reserved_keyword
  {
    $$ = NewColIdent(string($1))
  }

table_id:
  ID
  {
    $$ = NewTableIdent(string($1))
  }
| non_reserved_keyword
  {
    $$ = NewTableIdent(string($1))
  }

reserved_table_id:
  table_id
| reserved_keyword
  {
    $$ = NewTableIdent(string($1))
  }

/*
  These are not all necessarily reserved in MySQL, but some are.

  These are more importantly reserved because they may conflict with our grammar.
  If you want to move one that is not reserved in MySQL (i.e. ESCAPE) to the
  non_reserved_keywords, you'll need to deal with any conflicts.

  Sorted alphabetically
*/
reserved_keyword:
  ADD
| AND
| AS
| ASC
| AUTO_INCREMENT
| BETWEEN
| BY
| CASE
| CREATE
| CROSS
| CURRENT_DATE
| CURRENT_TIME
| CURRENT_TIMESTAMP
| DEFAULT
| DELETE
| DESC
| DESCRIBE
| DISTINCT
| DIV
| DROP
| ELSE
| END
| ESCAPE
| EXISTS
| FALSE
| FROM
| GROUP
| HAVING
| IF
| IGNORE
| IN
| INDEX
| INNER
| INSERT
| INTERVAL
| INTO
| IS
| JOIN
| KEY
| LEFT
| LIKE
| LIMIT
| MOD
| NATURAL
| NOT
| NULL
| ON
| OR
| ORDER
| OUTER
| REGEXP
| RENAME
| REPLACE
| RIGHT
| SELECT
| SEPARATOR
| SET
| SHOW
| TABLE
| TABLES
| THEN
| TO
| TRUE
| UNION
| UNIQUE
| UPDATE
| USING
| VALUES
| WHEN
| WHERE
| EXPLAIN

/*
  These are non-reserved Vitess, because they don't cause conflicts in the grammar.
  Some of them may be reserved in MySQL. The good news is we backtick quote them
  when we rewrite the query, so no issue should arise.

  Sorted alphabetically
*/
non_reserved_keyword:
  BIGINT
| BLOB
| BOOL
| CHAR
| DATE
| DATETIME
| DECIMAL
| DOUBLE
| FLOAT_TYPE
| FOREIGN
| INT
| INTEGER
| LAST_INSERT_ID
| LONGBLOB
| LONGTEXT
| MEDIUMBLOB
| MEDIUMINT
| MEDIUMTEXT
| NCHAR
| NUMERIC
| OFFSET
| PRIMARY
| REAL
| SIGNED
| SMALLINT
| TEXT
| TIME
| TIMESTAMP
| TINYBLOB
| TINYINT
| TINYTEXT
| UNSIGNED
| UNUSED
| VARCHAR
| YEAR
| ZEROFILL

openb:
  '('
  {
    if incNesting(yylex) {
      yylex.Error("max nesting level reached")
      return 1
    }
  }

closeb:
  ')'
  {
    decNesting(yylex)
  }

force_eof:
  {
    forceEOF(yylex)
  }

ddl_force_eof:
  {
    forceEOF(yylex)
  }
| openb
  {
    forceEOF(yylex)
  }
| reserved_sql_id
  {
    forceEOF(yylex)
  }
