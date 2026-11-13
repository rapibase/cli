package cli

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rapibase/cli/internal/i18n"
)

// tableInfo is what list_tables and describe_table return per table.
type tableInfo struct {
	Name       string `json:"name"`
	RowCount   int64  `json:"row_count"`
	PrimaryKey string `json:"primary_key"`
	Columns    []struct {
		Name         string  `json:"name"`
		Type         string  `json:"type"`
		Nullable     bool    `json:"nullable"`
		DefaultValue *string `json:"default_value"`
		IsPrimaryKey bool    `json:"is_primary_key"`
		References   *string `json:"references"`
	} `json:"columns"`
}

// tablesCmd reads the schema (MCP list_tables / describe_table). Unlike
// `sql`, a read-only session can use it (L4).
func (a *app) tablesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tables [table]",
		Short:   i18n.T("cmd.tables.short"),
		Long:    i18n.T("cmd.tables.long"),
		Args:    cobra.MaximumNArgs(1),
		Example: examples("rapibase tables", "rapibase tables orders", "rapibase tables orders --json"),
	}
	cmd.RunE = a.run(func(cmd *cobra.Command, args []string) (any, error) {
		t, err := a.resolve(false)
		if err != nil {
			return nil, err
		}
		path := "/api/projects/" + t.IDString() + "/tables"
		if len(args) == 1 {
			path += "/" + url.PathEscape(args[0])
		}
		var out struct {
			Result json.RawMessage `json:"result"`
		}
		if err := a.api.Do(a.ctx, "GET", path, nil, &out); err != nil {
			return nil, err
		}
		if len(args) == 1 {
			var ti tableInfo
			if json.Unmarshal(out.Result, &ti) != nil || ti.Name == "" {
				a.printRaw(out.Result)
			} else {
				a.printTableSchema(ti)
			}
			return map[string]any{"project": t.view(), "table": out.Result}, nil
		}
		var list struct {
			Tables []tableInfo `json:"tables"`
		}
		if json.Unmarshal(out.Result, &list) != nil {
			a.printRaw(out.Result)
			return map[string]any{"project": t.view(), "tables": out.Result}, nil
		}
		if len(list.Tables) == 0 {
			a.ui.Println(i18n.T("tables.none"))
		} else {
			rows := []map[string]any{}
			for _, ti := range list.Tables {
				cols := []string{}
				for _, c := range ti.Columns {
					cols = append(cols, c.Name)
				}
				rows = append(rows, map[string]any{
					"table": ti.Name, "rows": ti.RowCount, "primary_key": ti.PrimaryKey, "columns": strings.Join(cols, ", "),
				})
			}
			a.printTable([]string{"table", "rows", "primary_key", "columns"}, rows)
		}
		var raw any
		_ = json.Unmarshal(out.Result, &raw)
		if m, ok := raw.(map[string]any); ok {
			return map[string]any{"project": t.view(), "tables": m["tables"]}, nil
		}
		return map[string]any{"project": t.view(), "tables": out.Result}, nil
	})
	return cmd
}

// printTableSchema shows one table's columns.
func (a *app) printTableSchema(ti tableInfo) {
	a.ui.Println(a.ui.Bold(ti.Name) + "  " + a.ui.Dim(i18n.T("tables.rows", ti.RowCount)))
	rows := []map[string]any{}
	for _, c := range ti.Columns {
		def := ""
		if c.DefaultValue != nil {
			def = *c.DefaultValue
		}
		ref := ""
		if c.References != nil {
			ref = "→ " + *c.References
		}
		null := ""
		if !c.Nullable {
			null = "not null"
		}
		key := ""
		if c.IsPrimaryKey {
			key = "PK"
		}
		rows = append(rows, map[string]any{"column": c.Name, "type": c.Type, "null": null, "default": def, "key": key, "references": ref})
	}
	a.printTable([]string{"column", "type", "null", "default", "key", "references"}, rows)
}
