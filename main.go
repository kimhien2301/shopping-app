package main

import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/gorilla/mux"
    "github.com/kimhien2301/go-htmx-shopping-app/pkg/handlers"
    "github.com/kimhien2301/go-htmx-shopping-app/pkg/repository"
    "html/template"
    "log"
    "net/http"
    "path/filepath"
)

const (
    templateDir = "./templates"
    staticDir   = "./static"
)

var (
    tmpl *template.Template
    db   *sql.DB
    fs   http.Handler

    repo    *repository.Repository
    handler *handlers.Handler
)

func init() {
    pattern := filepath.Join(templateDir, "**", "*.html")
    tmpl = template.Must(template.ParseGlob(pattern))

    fs = http.FileServer(http.Dir(staticDir))

    initDB()
    repo = repository.NewRepository(db)
    handler = handlers.NewHandler(repo, tmpl, staticDir+"/uploads")
}

func initDB() {
    var err error

    db, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3333)/shopping?parseTime=true")
    if err != nil {
        log.Fatal(err)
    }

    if err = db.Ping(); err != nil {
        log.Fatal(err)
    }
}

func main() {
    defer func(db *sql.DB) {
        err := db.Close()
        if err != nil {
            log.Fatal(err)
        }
    }(db)

    router := mux.NewRouter()

    router.PathPrefix("/static").Handler(http.StripPrefix("/static", fs))

    // Admin routes
    router.HandleFunc("/seed-products", handler.SeedProduct).Methods("POST")
    router.HandleFunc("/manage-products", handler.ProductsPage).Methods("GET")
    router.HandleFunc("/all-products", handler.AllProductsView).Methods("GET")
    router.HandleFunc("/products", handler.ListProducts).Methods("GET")
    router.HandleFunc("/product/{id}", handler.ProductView).Methods("GET")
    router.HandleFunc("/create-product", handler.CreateProductView).Methods("GET")
    router.HandleFunc("/product", handler.CreateProduct).Methods("POST")
    router.HandleFunc("/product/{id}", handler.DeleteProduct).Methods("DELETE")
    router.HandleFunc("/product/{id}/edit", handler.EditProductView).Methods("GET")
    router.HandleFunc("/product/{id}", handler.EditProduct).Methods("PUT")

    router.HandleFunc("/manage-orders", handler.ManageOrdersPage).Methods("GET")
    router.HandleFunc("/order-table", handler.OrderTableView).Methods("GET")
    router.HandleFunc("/orders", handler.OrderTableRowsView).Methods("GET")
    router.HandleFunc("/order/{id}", handler.OrderDetailView).Methods("GET")
    router.HandleFunc("/order/{id}", handler.UpdateOrderStatus).Methods("PUT")

    // User shopping routes
    router.HandleFunc("/", handler.ShoppingHomePage).Methods("GET")
    router.HandleFunc("/shopping-items", handler.ShoppingItemView).Methods("GET")
    router.HandleFunc("/cart-items", handler.CartItemView).Methods("GET")
    router.HandleFunc("/cart/add/{id}", handler.AddCartItem).Methods("POST")
    router.HandleFunc("/shopping-cart", handler.ShoppingCartView).Methods("GET")
    router.HandleFunc("/cart/{id}", handler.ShoppingCartUpdate).Methods("PUT")
    router.HandleFunc("/order-complete", handler.PlaceOrder)

    // Logger
    router.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
            next.ServeHTTP(w, r)
        })
    })

    err := http.ListenAndServe(":5000", router)
    if err != nil {
        log.Fatal(err)
    }
}
