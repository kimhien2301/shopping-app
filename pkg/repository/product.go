package repository

import (
    "database/sql"
    "github.com/google/uuid"
    . "github.com/kimhien2301/go-htmx-shopping-app/pkg/models"
    "time"
)

type ProductRepository struct {
    db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
    return &ProductRepository{
        db: db,
    }
}

func (r *ProductRepository) Create(product *Product) (*Product, error) {
    query := "insert into products(id, name, price, description, image) values (?, ?, ?, ?, ?)"
    stmt, err := r.db.Prepare(query)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()

    newId, err := uuid.NewUUID()
    if err != nil {
        return nil, err
    }

    product.ID = newId

    result, err := stmt.Exec(product.ID, product.Name, product.Price, product.Description, product.Image)
    if err != nil {
        return nil, err
    }

    if rowAffected, err := result.RowsAffected(); err != nil || rowAffected == 0 {
        return nil, sql.ErrNoRows
    }

    return product, nil
}

func (r *ProductRepository) GetById(id uuid.UUID) (*Product, error) {
    query := "select id, name, price, description, image, created_date, modified_date from products where id = ?"
    row := r.db.QueryRow(query, id)

    product := &Product{}
    err := row.Scan(&product.ID, &product.Name, &product.Price, &product.Description, &product.Image, &product.CreatedDate, &product.ModifiedDate)
    if err != nil {
        return nil, err
    }

    return product, nil
}

func (r *ProductRepository) Update(id uuid.UUID, product *Product) error {
    query := "update products set name = ?, price = ?, description = ?, image = ?, modified_date = ? where id = ?"
    stmt, err := r.db.Prepare(query)
    if err != nil {
        return err
    }

    _, err = stmt.Exec(product.Name, product.Price, product.Description, product.Image, time.Now(), id)
    return err
}

func (r *ProductRepository) Delete(id uuid.UUID) error {
    _, err := r.db.Exec("delete from products where id = ?", id)
    return err
}

func (r *ProductRepository) Find(whereClause string, page, size int) ([]Product, error) {
    var products []Product

    query := `select id, name, price, description, image, created_date, modified_date from products`

    if whereClause != "" {
        query += " where " + whereClause
    }

    query += " order by created_date desc"

    var rows *sql.Rows
    var err error
    if size != -1 {
        query += " limit ? offset ?"
        rows, err = r.db.Query(query, size, (page-1)*size)
    } else {
        rows, err = r.db.Query(query)
    }
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var product Product

        err := rows.Scan(&product.ID, &product.Name, &product.Price, &product.Description, &product.Image, &product.CreatedDate, &product.ModifiedDate)
        if err != nil {
            return nil, err
        }

        products = append(products, product)
    }

    return products, nil
}

func (r *ProductRepository) Count() (int, error) {
    var count int
    err := r.db.QueryRow("select count(*) from products").Scan(&count)
    if err != nil {
        return 0, err
    }
    return count, nil
}
