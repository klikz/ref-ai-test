package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

func writeoffItemKey(lineID int, serial string) string {
	return fmt.Sprintf("%d:%s", lineID, strings.TrimSpace(serial))
}

func writeoffComponentKey(lineID, componentID int) string {
	return fmt.Sprintf("%d:%d", lineID, componentID)
}

func (r *Repo) writeoffPrefetchProductModels(items []WriteoffSaveItemInput) (map[string]int, error) {
	lineSerials := map[int][]string{}
	for _, item := range items {
		if !IsWriteoffProductLine(item.LineID) {
			continue
		}
		serial := strings.TrimSpace(item.Serial)
		if serial == "" {
			continue
		}
		lineSerials[item.LineID] = append(lineSerials[item.LineID], serial)
	}

	result := map[string]int{}
	for lineID, serials := range lineSerials {
		if len(serials) == 0 {
			continue
		}
		rows, err := r.store.db.Query(`
			SELECT serial, model_id, status
			FROM lines.products
			WHERE line_id = $1 AND serial = ANY($2)`,
			lineID, pq.Array(serials),
		)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var serial, status string
			var modelID int
			if err := rows.Scan(&serial, &modelID, &status); err != nil {
				rows.Close()
				return nil, err
			}
			if status != ProductStatusActive {
				rows.Close()
				return nil, fmt.Errorf("serial %s active emas", serial)
			}
			if err := r.writeoffModelAllowedOnLine(lineID, modelID); err != nil {
				rows.Close()
				return nil, err
			}
			result[writeoffItemKey(lineID, serial)] = modelID
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return result, nil
}

func (r *Repo) writeoffPrefetchComponentBalances(items []WriteoffSaveItemInput) (map[string]float64, error) {
	type pair struct {
		lineID      int
		componentID int
	}
	seen := map[pair]struct{}{}
	pairs := make([]pair, 0)
	for _, item := range items {
		if !IsWriteoffAuxLine(item.LineID) || item.ComponentID <= 0 {
			continue
		}
		p := pair{lineID: item.LineID, componentID: item.ComponentID}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		pairs = append(pairs, p)
	}
	result := map[string]float64{}
	if len(pairs) == 0 {
		return result, nil
	}

	lineIDs := make([]int, len(pairs))
	componentIDs := make([]int, len(pairs))
	for i, p := range pairs {
		lineIDs[i] = p.lineID
		componentIDs[i] = p.componentID
	}

	rows, err := r.store.db.Query(`
		SELECT b.line_id, b.component_id, COALESCE(b.quantity, 0)
		FROM lines.balance b
		INNER JOIN unnest($1::int[], $2::int[]) AS pairs(line_id, component_id)
			ON b.line_id = pairs.line_id AND b.component_id = pairs.component_id`,
		pq.Array(lineIDs), pq.Array(componentIDs),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var lineID, componentID int
		var qty float64
		if err := rows.Scan(&lineID, &componentID, &qty); err != nil {
			return nil, err
		}
		result[writeoffComponentKey(lineID, componentID)] = roundWareQuantity(qty)
	}
	return result, rows.Err()
}

func writeoffDocumentItemsInsertBatch(tx *sql.Tx, documentID int64, items []WriteoffSaveItemInput) error {
	if len(items) == 0 {
		return nil
	}
	var sb strings.Builder
	args := make([]any, 0, len(items)*9+1)
	args = append(args, documentID)
	sb.WriteString(`INSERT INTO writeoff.document_items
		(document_id, line_id, item_type, model_id, component_id, serial, quantity, comment, sort_order) VALUES `)
	for i, item := range items {
		if i > 0 {
			sb.WriteString(",")
		}
		n := i*9 + 2
		sb.WriteString(fmt.Sprintf("($1,$%d,$%d,NULLIF($%d,0),NULLIF($%d,0),$%d,$%d,$%d,$%d)",
			n, n+1, n+2, n+3, n+4, n+5, n+6, n+7))
		qty := roundWareQuantity(item.Quantity)
		args = append(args,
			item.LineID, item.ItemType, item.ModelID, item.ComponentID,
			strings.TrimSpace(item.Serial), qty, strings.TrimSpace(item.Comment), item.SortOrder,
		)
	}
	_, err := tx.Exec(sb.String(), args...)
	return err
}
