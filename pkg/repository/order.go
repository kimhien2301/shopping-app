package repository

import (
    "database/sql"
    "github.com/google/uuid"
    . "github.com/kimhien2301/go-htmx-shopping-app/pkg/models"
)

type OrderRepository struct {
    db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
    return &OrderRepository{
        db: db,
    }
}

func (r *OrderRepository) PlaceOrderWithItems(orderItems []OrderItem) error {
    tx, err := r.db.Begin()
    if err != nil {
        return err
    }

    order := &Order{
        ID:     uuid.Must(uuid.NewV7()),
        UserID: "kimhien2301@gmail.com",
        Status: Ordered,
        Items:  orderItems,
    }

    query := "insert into orders(id, user_id, status) values (?, ?, ?)"
    result, err := tx.Exec(query, order.ID, order.UserID, order.Status)
    if err != nil {
        tx.Rollback()
        return err
    }

    if rowAffected, err := result.RowsAffected(); err != nil || rowAffected == 0 {
        return sql.ErrNoRows
    }

    for _, item := range orderItems {
        query = "insert into order_items(order_id, product_id, quantity, cost) values (?, ?, ?, ?)"
        _, err = tx.Exec(query, order.ID, item.ProductID, item.Quantity, item.Cost)
        if err != nil {
            tx.Rollback()
            return err
        }
    }

    err = tx.Commit()
    if err != nil {
        return err
    }

    return nil
}

func (r *OrderRepository) GetById(id uuid.UUID) (*Order, error) {
    query := "select id, user_id, status, date from orders where id = ?"
    row := r.db.QueryRow(query, id)

    order := &Order{}
    err := row.Scan(&order.ID, &order.UserID, &order.Status, &order.Date)
    if err != nil {
        return nil, err
    }

    return order, nil
}

func (r *OrderRepository) Update(id uuid.UUID, order *Order) error {
    query := "update orders set status = ?, date = ? where id = ?"
    stmt, err := r.db.Prepare(query)
    if err != nil {
        return err
    }

    _, err = stmt.Exec(order.Status, order.Date, id)
    return err
}

func (r *OrderRepository) UpdateStatus(id uuid.UUID, status OrderStatus) error {
    query := "update orders set status = ? where id = ?"
    stmt, err := r.db.Prepare(query)
    if err != nil {
        return err
    }

    _, err = stmt.Exec(status, id)
    return err
}

func (r *OrderRepository) Delete(id uuid.UUID) error {
    _, err := r.db.Exec("delete from orders where id = ?", id)
    return err
}

func (r *OrderRepository) Find(page, size int) ([]Order, error) {
    var orders []Order

    query := `
            with result as (
                    select id, user_id, status, date from orders limit ? offset ?
                ),
                cost(id, total) as (
                    select r.id, sum(oi.cost)
                    from result r
                        inner join order_items oi on r.id = oi.order_id
                    group by r.id
                )
                select r.id, r.user_id, r.status, r.date, coalesce(c.total, 0)
                from result r
                    left join cost c on r.id = c.id
            `
    rows, err := r.db.Query(query, size, (page-1)*size)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var order Order

        err := rows.Scan(&order.ID, &order.UserID, &order.Status, &order.Date, &order.Total)
        if err != nil {
            return nil, err
        }

        orders = append(orders, order)
    }

    return orders, nil
}

func (r *OrderRepository) Count() (int, error) {
    var count int
    err := r.db.QueryRow("select count(*) from orders").Scan(&count)
    if err != nil {
        return 0, err
    }
    return count, nil
}

func (r *OrderRepository) GetOrderWithProduct(id uuid.UUID) (*Order, error) {
    query := "select id, user_id, status, date from orders where id = ?"
    row := r.db.QueryRow(query, id)

    order := &Order{}
    err := row.Scan(&order.ID, &order.UserID, &order.Status, &order.Date)
    if err != nil {
        return nil, err
    }

    itemsQuery := `
                    select i.product_id, i.quantity, i.cost, p.name, p.price, p.description, p.image, p.created_date, p.modified_date
                    from order_items i
                    inner join products p on p.id = i.product_id
                    where order_id = ?
                    `

    rows, err := r.db.Query(itemsQuery, id)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var item OrderItem
        var product Product
        err := rows.Scan(
            &item.ProductID,
            &item.Quantity,
            &item.Cost,
            &product.Name,
            &product.Price,
            &product.Description,
            &product.Image,
            &product.CreatedDate,
            &product.ModifiedDate,
        )
        if err != nil {
            return nil, err
        }

        item.OrderID = id
        item.Product = &product
        //item.Cost = float64(item.Quantity) * item.Product.Price
        //item.Product.ID = item.ProductID

        order.Items = append(order.Items, item)
    }

    for _, item := range order.Items {
        order.Total += item.Cost
    }

    return order, nil
}
