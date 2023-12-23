package pkg

import (
	"database/sql"
	"errors"
	"fmt"
	"go/format"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const (
	golangBool      = "bool"
	sqlNullBool     = "*bool"
	golangInt32     = "int32"
	sqlNullInt32    = "*int32"
	golangInt64     = "int64"
	sqlNullInt64    = "*int64"
	golangFloat32   = "float32"
	golangFloat64   = "float64"
	sqlNullFloat    = "*float64"
	golangString    = "string"
	sqlNullString   = "*string"
	golangTime      = "time.Time"
	sqlNullTime     = "*time.Time"
	golangByteArray = "[]byte"

	packageDatabaseSql = ""
	packageTime        = "time"

	permissionReadWrite = 0644
)

var _pluralize = NewClient()

type ColumnInfo struct {
	DefaultValue   sql.NullString
	Nullable       sql.NullString
	DataType       string
	ColumnType     string
	ColumnKey      sql.NullString
	Extra          sql.NullString
	MappedDataType string
}

func DB2Struct(dsnDb string, database string, tableNames []string, outputFolder, packageName string) (err error) {
	var db *sql.DB
	db, err = sql.Open("mysql", dsnDb)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, tableName := range tableNames {
		if err = generateTable(database, tableName, outputFolder, packageName, db); err != nil {
			log.Println(err, "Error generate struct of table", "table name", tableName)
			continue
		}
	}
	return nil
}

func generateTable(databaseName string, tableName string, outputFolder, packageName string, db *sql.DB) error {
	rows, err := db.Query("SELECT COLUMN_NAME, COLUMN_DEFAULT, IS_NULLABLE, DATA_TYPE, COLUMN_TYPE, COLUMN_KEY, EXTRA FROM information_schema.columns WHERE TABLE_NAME = ? AND TABLE_SCHEMA = ? ORDER BY ORDINAL_POSITION", tableName, databaseName)
	if err != nil {
		return err
	}
	defer rows.Close()

	var columnNamesSorted []string
	column2Info := make(map[string]*ColumnInfo)

	for rows.Next() {
		var name string
		var defaultValue sql.NullString
		var nullable sql.NullString
		var dataType string
		var columnType string
		var key sql.NullString
		var extra sql.NullString
		err = rows.Scan(&name, &defaultValue, &nullable, &dataType, &columnType, &key, &extra)
		if err != nil {
			return err
		}
		columnNamesSorted = append(columnNamesSorted, name)

		column2Info[name] = &ColumnInfo{
			DefaultValue: defaultValue,
			Nullable:     nullable,
			DataType:     dataType,
			ColumnType:   columnType,
			ColumnKey:    key,
			Extra:        extra,
		}
	}

	if len(columnNamesSorted) == 0 {
		return errors.New("no results returned for table")
	}

	src, err := generate(column2Info, columnNamesSorted, tableName, packageName)

	if err != nil {
		return err
	}
	//write output
	fileName := outputFolder + _pluralize.Singular(tableName) + ".table.go"
	file, err := os.OpenFile(fileName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, permissionReadWrite)
	if err != nil {
		return err
	}
	_, err = file.WriteString(string(src))
	return err
}

func generate(column2Info map[string]*ColumnInfo, columnNames []string, tableName, packageName string) ([]byte, error) {
	dbTypes, mapPackage, mapFieldName2Column, mapDataType, fieldNameSorted := generateMysqlTypes(column2Info, columnNames)
	structName := _pluralize.Singular(convertName(tableName))
	imports := generatePackage(mapPackage)

	src := "// auto generated, don't change it, create *.custom.go file for custom the data\n"
	src += "\n"
	if len(imports) == 0 {
		src += fmt.Sprintf("package %s\ntype %s %s", packageName, structName, dbTypes)
	} else {
		src += fmt.Sprintf("package %s\n%s\ntype %s %s", packageName, imports, structName, dbTypes)
	}

	alias := strings.ToLower(string(structName[0]))
	tableNameFunc := "func (" + alias + " " + structName + ") TableName() string {\n" +
		"	return \"" + tableName + "\"" +
		"}"

	src += fmt.Sprintf("\n%s", tableNameFunc)

	generateColumn := generateColumnConst(mapFieldName2Column, fieldNameSorted, structName)
	src += fmt.Sprintf("\n%s", generateColumn)

	src += "\n"
	t := generateGetters(mapDataType, structName, alias)
	src += t
	src += "\n"
	formatted, err := format.Source([]byte(src))
	if err != nil {
		err = fmt.Errorf("error formatting: %s, was formatting\n%s", err, src)
	}

	return formatted, err
}

func generatePackage(mapPackage map[string]bool) string {
	listPackage := make([]string, 0, len(mapPackage))
	for key := range mapPackage {
		listPackage = append(listPackage, fmt.Sprintf("\"%s\"", key))
	}

	if len(listPackage) == 0 {
		return ""
	}

	return fmt.Sprintf("import(\n%s\n)", strings.Join(listPackage, "\n"))
}

func generateColumnConst(mapFieldName2Column map[string]string, fieldNameSorted []string, structName string) string {
	var res string
	typeName := fmt.Sprintf("%s%sColumn", strings.ToLower(structName[0:1]), structName[1:])
	res = fmt.Sprintf("type %s struct {", typeName)
	for _, colName := range fieldNameSorted {
		res += fmt.Sprintf("\n%s string", colName)
	}
	res += "\n}"

	res += fmt.Sprintf("\nvar Map%sColumn = %s{", structName, typeName)
	for _, colName := range fieldNameSorted {
		res += fmt.Sprintf("\n%s: \"%s\",", colName, mapFieldName2Column[colName])
	}
	res += "\n}"
	return res
}

func generateMysqlTypes(column2Info map[string]*ColumnInfo, columnNames []string) (string, map[string]bool, map[string]string, map[string]string, []string) {
	structure := "struct {"
	mapPackage := make(map[string]bool)
	mapFieldName2Column := make(map[string]string)
	mapDataType := make(map[string]string)
	fieldNameSorted := make([]string, 0, len(columnNames))

	for _, colName := range columnNames {
		info := column2Info[colName]
		nullable := false
		if info.Nullable.String == "YES" {
			nullable = true
		}
		valueType := mysqlType2GoType(info.DataType, nullable, mapPackage)

		fieldName := convertName(colName)
		mapDataType[fieldName] = valueType
		mapFieldName2Column[fieldName] = colName
		fieldNameSorted = append(fieldNameSorted, fieldName)

		var annotations []string
		if info.ColumnKey.String == "PRI" {
			annotations = append(annotations, "primary_key")
		}
		if info.Extra.String != "" {
			annotations = append(annotations, info.Extra.String)
		}
		annotations = append(annotations, fmt.Sprintf("type:%s", info.ColumnType))
		if info.DefaultValue.String != "" {
			annotations = append(annotations, fmt.Sprintf("default:%s", info.DefaultValue.String))
		}

		structure += fmt.Sprintf("\n%s %s `gorm:\"%s\"`", fieldName, valueType, strings.Join(annotations, ";"))
	}

	structure += "\n}"
	return structure, mapPackage, mapFieldName2Column, mapDataType, fieldNameSorted
}

func generateGetters(column2Info map[string]string, tableName string, alias string) string {
	result := "\n"
	for colName, dataType := range column2Info {
		result += fmt.Sprintf("\nfunc (%s *%s) Get%s() %s {\n\treturn %s.%s\n}", alias, tableName, colName, dataType, alias, colName)
		result += "\n"
	}
	return result
}

func mysqlType2GoType(mysqlType string, nullable bool, mapPackage map[string]bool) string {
	switch mysqlType {
	case "tinyint":
		if nullable {
			//mapPackage[packageDatabaseSql] = true
			return sqlNullBool
		} else {
			return golangBool
		}
	case "int", "smallint", "mediumint":
		if nullable {
			//mapPackage[packageDatabaseSql] = true
			return sqlNullInt32
		} else {
			return golangInt32
		}
	case "bigint":
		if nullable {
			//mapPackage[packageDatabaseSql] = true
			return sqlNullInt64
		} else {
			return golangInt64
		}
	case "float":
		if nullable {
			//mapPackage[packageDatabaseSql] = true
			return sqlNullFloat
		} else {
			return golangFloat32
		}
	case "double", "decimal":
		if nullable {
			//mapPackage[packageDatabaseSql] = true
			return sqlNullFloat
		} else {
			return golangFloat64
		}
	case "char", "varchar", "nvarchar", "enum", "text", "tinytext", "mediumtext", "longtext", "json":
		if nullable {
			//mapPackage[packageDatabaseSql] = true
			return sqlNullString
		} else {
			return golangString
		}
	case "date", "datetime", "time", "timestamp":
		if nullable {
			//mapPackage[packageDatabaseSql] = true
			return sqlNullTime
		} else {
			mapPackage[packageTime] = true
			return golangTime
		}
	case "binary", "verbinary", "blob", "mediumblob", "longblob":
		return golangByteArray
	default:
		return ""
	}
}

// covert snake case to camel case
func convertName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}

	n := strings.Builder{}
	n.Grow(len(name))
	capNext := true
	for _, v := range []byte(name) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if capNext && vIsLow {
			v += 'A'
			v -= 'a'
		}
		if vIsCap || vIsLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
		}
	}
	return n.String()
}
