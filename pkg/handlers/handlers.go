package handlers

import (
    "errors"
    "fmt"
    "github.com/bxcodec/faker/v3"
    "github.com/google/uuid"
    "github.com/gorilla/mux"
    . "github.com/kimhien2301/go-htmx-shopping-app/pkg/models"
    "github.com/kimhien2301/go-htmx-shopping-app/pkg/repository"
    "golang.org/x/text/cases"
    "golang.org/x/text/language"
    "html/template"
    "io"
    "log"
    "math"
    "math/rand"
    "net/http"
    "os"
    "path/filepath"
    "slices"
    "strconv"
    "time"
)

type Handler struct {
    Repo             *repository.Repository
    Tmpl             *template.Template
    ImageStoragePath string
}

func NewHandler(repo *repository.Repository, tmpl *template.Template, fp string) *Handler {
    return &Handler{
        Repo:             repo,
        Tmpl:             tmpl,
        ImageStoragePath: fp,
    }
}

/*** Admin actions ***/

func (h *Handler) SeedProduct(w http.ResponseWriter, r *http.Request) {
    src := rand.NewSource(time.Now().UnixNano())
    rnd := rand.New(src)

    numProducts := 50

    productTypes := []string{"Laptop", "Smartphone", "Tablet", "Headphones", "Speaker", "Camera", "TV", "Watch", "Printer", "Monitor"}

    for i := 0; i < numProducts; i++ {
        productType := productTypes[rnd.Intn(len(productTypes))]
        caser := cases.Title(language.English)
        productName := caser.String(faker.Word()) + " " + productType

        id, err := uuid.NewV7()
        if err != nil {
            http.Error(w, fmt.Sprintf("Error creating product %s: %v", productName, err), http.StatusInternalServerError)
            return
        }

        product := Product{
            ID:          id,
            Name:        productName,
            Price:       float64(rnd.Intn(10000)) / 100,
            Description: faker.Sentence(),
            Image:       "placeholder.jpg",
        }

        _, err = h.Repo.Product.Create(&product)
        if err != nil {
            http.Error(w, fmt.Sprintf("Error creating product %s: %v", productName, err), http.StatusInternalServerError)
            return
        }
    }

    w.WriteHeader(http.StatusCreated)
    fmt.Fprintf(w, "Successfully seeded %d dummy products", numProducts)
}

func (h *Handler) ProductsPage(w http.ResponseWriter, r *http.Request) {
    err := h.Tmpl.ExecuteTemplate(w, "products", nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) AllProductsView(w http.ResponseWriter, r *http.Request) {
    err := h.Tmpl.ExecuteTemplate(w, "allProducts", nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
    pageStr := r.URL.Query().Get("page")
    sizeStr := r.URL.Query().Get("size")

    page, err := strconv.Atoi(pageStr)
    if err != nil || page < 1 {
        page = 1
    }

    size, err := strconv.Atoi(sizeStr)
    if err != nil || size <= 0 {
        size = 10
    }

    products, err := h.Repo.Product.Find("", page, size)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    count, err := h.Repo.Product.Count()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    totalPages := int(math.Ceil(float64(count) / float64(size)))
    previousPage := page - 1
    nextPage := page + 1
    pageRange := makeRange(1, totalPages)

    data := struct {
        Products     []Product
        CurrentPage  int
        TotalPages   int
        Size         int
        PreviousPage int
        NextPage     int
        PageRange    []int
    }{
        Products:     products,
        CurrentPage:  page,
        TotalPages:   totalPages,
        Size:         size,
        PreviousPage: previousPage,
        NextPage:     nextPage,
        PageRange:    pageRange,
    }

    //time.Sleep(3 * time.Second)

    err = h.Tmpl.ExecuteTemplate(w, "productRows", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) ProductView(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := uuid.Parse(vars["id"])
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    product, err := h.Repo.Product.GetById(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    err = h.Tmpl.ExecuteTemplate(w, "viewProduct", product)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) CreateProductView(w http.ResponseWriter, r *http.Request) {
    err := h.Tmpl.ExecuteTemplate(w, "createProduct", nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
    var errorMessages []string
    //TODO: add observer pattern to handle errors

    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Failed to parse form", http.StatusInternalServerError)
        log.Println(err)
        return
    }

    nameValue := r.FormValue("name")
    priceValue := r.FormValue("price")
    descriptionValue := r.FormValue("description")

    if nameValue == "" {
        errorMessages = append(errorMessages, "Product name is required.")
    }

    if priceValue == "" {
        errorMessages = append(errorMessages, "Product price is required.")
    }

    priceParsed, err := strconv.ParseFloat(priceValue, 64)
    if err != nil {
        errorMessages = append(errorMessages, "Invalid price value.")
    }

    if len(errorMessages) > 0 {
        sendMessage(w, h.Tmpl, errorMessages, nil)
        return
    }

    // Process image upload
    filename := ""

    file, handler, err := r.FormFile("image")
    if err != nil && !errors.Is(err, http.ErrMissingFile) {
        errorMessages = append(errorMessages, "Error retrieving the image")
        sendMessage(w, h.Tmpl, errorMessages, nil)
        return
    } else if !errors.Is(err, http.ErrMissingFile) {
        defer file.Close()

        nano := strconv.FormatInt(time.Now().UnixNano(), 10)
        filename = nano + filepath.Ext(handler.Filename)

        filePath := filepath.Join(h.ImageStoragePath, filename)

        dst, err := os.Create(filePath)
        if err != nil {
            errorMessages = append(errorMessages, "Error saving new file: "+err.Error())
            sendMessage(w, h.Tmpl, errorMessages, nil)
            return
        }
        defer dst.Close()

        if _, err := io.Copy(dst, file); err != nil {
            errorMessages = append(errorMessages, "Error saving new file: "+err.Error())
            sendMessage(w, h.Tmpl, errorMessages, nil)
            return
        }
    }

    id, err := uuid.NewV7()
    if err != nil {
        errorMessages = append(errorMessages, "Error get new id: "+err.Error())
        sendMessage(w, h.Tmpl, errorMessages, nil)
        return
    }

    product := &Product{
        ID:          id,
        Name:        nameValue,
        Price:       priceParsed,
        Description: descriptionValue,
        Image:       filename,
    }

    product, err = h.Repo.Product.Create(product)
    if err != nil {
        errorMessages = append(errorMessages, "Failed to create product: "+err.Error())
        sendMessage(w, h.Tmpl, errorMessages, nil)
        return
    }

    sendMessage(w, h.Tmpl, nil, product)
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := uuid.Parse(vars["id"])
    if err != nil {
        http.Error(w, "Invalid product ID", http.StatusBadRequest)
        return
    }

    product, err := h.Repo.Product.GetById(id)
    if err != nil {
        http.Error(w, "Product not found", http.StatusInternalServerError)
        return
    }

    err = h.Repo.Product.Delete(id)
    if err != nil {
        http.Error(w, "Error deleting product", http.StatusInternalServerError)
        return
    }

    if product.Image != "" {
        imagePath := filepath.Join(h.ImageStoragePath, product.Image)
        err = os.Remove(imagePath)
        if err != nil {
            http.Error(w, "Error deleting product image", http.StatusInternalServerError)
            return
        }
    }

    err = h.Tmpl.ExecuteTemplate(w, "allProducts", nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) EditProductView(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := uuid.Parse(vars["id"])
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    product, err := h.Repo.Product.GetById(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    err = h.Tmpl.ExecuteTemplate(w, "editProduct", product)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) EditProduct(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := uuid.Parse(vars["id"])
    if err != nil {
        http.Error(w, "Invalid product ID", http.StatusBadRequest)
        return
    }

    var errorMessages []string
    //TODO: add observer pattern to handle errors

    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Failed to parse form", http.StatusInternalServerError)
        log.Println(err)
        return
    }

    nameValue := r.FormValue("name")
    priceValue := r.FormValue("price")
    descriptionValue := r.FormValue("description")

    if nameValue == "" {
        errorMessages = append(errorMessages, "Product name is required.")
    }

    if priceValue == "" {
        errorMessages = append(errorMessages, "Product price is required.")
    }

    priceParsed, err := strconv.ParseFloat(priceValue, 64)
    if err != nil {
        errorMessages = append(errorMessages, "Invalid price value.")
    }

    if len(errorMessages) > 0 {
        sendMessage(w, h.Tmpl, errorMessages, nil)
        return
    }

    product := &Product{
        Name:        nameValue,
        Price:       priceParsed,
        Description: descriptionValue,
    }

    err = h.Repo.Product.Update(id, product)
    if err != nil {
        errorMessages = append(errorMessages, "Failed to update product: "+err.Error())
        sendMessage(w, h.Tmpl, errorMessages, nil)
        return
    }

    sendMessage(w, h.Tmpl, nil, product)
}

/*** End Admin actions ***/

/*** User actions ***/
var (
    currentCartOrderId uuid.UUID
    cartItems          []OrderItem
)

func (h *Handler) ShoppingHomePage(w http.ResponseWriter, r *http.Request) {
    data := struct {
        OrderItems []OrderItem
    }{
        OrderItems: cartItems,
    }

    err := h.Tmpl.ExecuteTemplate(w, "homepage", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func (h *Handler) ShoppingItemView(w http.ResponseWriter, r *http.Request) {
    time.Sleep(1 * time.Second)

    products, _ := h.Repo.Product.Find("", 1, -1)
    err := h.Tmpl.ExecuteTemplate(w, "shoppingItems", products)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func (h *Handler) CartItemView(w http.ResponseWriter, r *http.Request) {
    data := &Cart{
        Items:     cartItems,
        Message:   "",
        AlertType: "",
        TotalCost: getCartTotal(),
    }

    err := h.Tmpl.ExecuteTemplate(w, "cartItems", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func (h *Handler) AddCartItem(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := uuid.Parse(vars["id"])
    if err != nil {
        http.Error(w, "Invalid product ID", http.StatusBadRequest)
        return
    }

    if currentCartOrderId == uuid.Nil {
        currentCartOrderId, _ = uuid.NewV7()
    }

    cartMessage := ""
    cartAlert := ""

    productExists := false
    for index, item := range cartItems {
        if item.ProductID == id {
            productExists = true
            cartItems[index].Quantity++
            cartItems[index].Cost = cartItems[index].Product.Price * float64(cartItems[index].Quantity)

            cartMessage = cartItems[index].Product.Name + " quantity updated."
            cartAlert = "warning"
            break
        }
    }

    if !productExists {
        product, err := h.Repo.Product.GetById(id)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        cartItems = append(cartItems, OrderItem{
            OrderID:   currentCartOrderId,
            ProductID: id,
            Product:   product,
            Quantity:  1,
            Cost:      product.Price,
        })

        cartMessage = product.Name + " added."
        cartAlert = "success"
    }

    data := &Cart{
        Items:     cartItems,
        Message:   cartMessage,
        AlertType: cartAlert,
        TotalCost: getCartTotal(),
    }

    err = h.Tmpl.ExecuteTemplate(w, "cartItems", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func (h *Handler) ShoppingCartView(w http.ResponseWriter, r *http.Request) {
    err := h.Tmpl.ExecuteTemplate(w, "shoppingCart", cartItems)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func (h *Handler) ShoppingCartUpdate(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := uuid.Parse(vars["id"])
    if err != nil {
        http.Error(w, "Invalid product ID", http.StatusBadRequest)
        return
    }

    index, item := getCartItem(id)
    if index == -1 {
        http.Error(w, "Product not found in order", http.StatusBadRequest)
        return
    }

    refreshCart := false

    action := r.URL.Query().Get("action")
    switch action {
    case "add":
        item.Quantity++
        item.Cost = item.Product.Price * float64(item.Quantity)
        break
    case "subtract":
        if item.Quantity > 1 {
            item.Quantity--
            item.Cost = item.Product.Price * float64(item.Quantity)
        } else {
            cartItems = append(cartItems[:index], cartItems[index+1:]...)
            refreshCart = true
        }
        break
    case "remove":
        cartItems = append(cartItems[:index], cartItems[index+1:]...)
        refreshCart = true
        break
    default:
        break
    }

    data := &Cart{
        Items:       cartItems,
        TotalCost:   getCartTotal(),
        AlertType:   "info",
        RefreshCart: refreshCart,
    }

    err = h.Tmpl.ExecuteTemplate(w, "updateShoppingCart", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
    if len(cartItems) == 0 {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    err := h.Repo.Order.PlaceOrderWithItems(cartItems)
    if err != nil {
        http.Error(w, "Failed to place order", http.StatusInternalServerError)
        return
    }

    orderItems := cartItems
    orderTotal := getCartTotal()

    // Empty the cart
    cartItems = []OrderItem{}
    currentCartOrderId = uuid.Nil

    data := &Cart{
        Items:     orderItems,
        TotalCost: orderTotal,
    }

    err = h.Tmpl.ExecuteTemplate(w, "orderComplete", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func (h *Handler) ManageOrdersPage(w http.ResponseWriter, r *http.Request) {
    err := h.Tmpl.ExecuteTemplate(w, "orders", nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func (h *Handler) OrderTableView(w http.ResponseWriter, r *http.Request) {
    err := h.Tmpl.ExecuteTemplate(w, "orderTable", nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) OrderTableRowsView(w http.ResponseWriter, r *http.Request) {
    time.Sleep(1 * time.Second)

    pageStr := r.URL.Query().Get("page")
    sizeStr := r.URL.Query().Get("size")

    page, err := strconv.Atoi(pageStr)
    if err != nil || page < 1 {
        page = 1
    }

    size, err := strconv.Atoi(sizeStr)
    if err != nil || size <= 0 {
        size = 10
    }

    orders, err := h.Repo.Order.Find(page, size)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    count, err := h.Repo.Order.Count()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    totalPages := int(math.Ceil(float64(count) / float64(size)))
    previousPage := page - 1
    nextPage := page + 1
    pageRange := makeRange(1, totalPages)

    data := struct {
        Orders       []Order
        CurrentPage  int
        TotalPages   int
        Size         int
        PreviousPage int
        NextPage     int
        PageRange    []int
    }{
        Orders:       orders,
        CurrentPage:  page,
        TotalPages:   totalPages,
        Size:         size,
        PreviousPage: previousPage,
        NextPage:     nextPage,
        PageRange:    pageRange,
    }

    err = h.Tmpl.ExecuteTemplate(w, "orderTableRows", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) OrderDetailView(w http.ResponseWriter, r *http.Request) {
    time.Sleep(1 * time.Second)

    vars := mux.Vars(r)
    id, err := uuid.Parse(vars["id"])
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    order, err := h.Repo.Order.GetOrderWithProduct(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    data := struct {
        Order         *Order
        StatusOptions []OrderStatus
    }{
        Order:         order,
        StatusOptions: []OrderStatus{Ordered, Pending, Shipped, Delivered, Cancel},
    }

    err = h.Tmpl.ExecuteTemplate(w, "orderDetail", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := uuid.Parse(vars["id"])
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    status := r.FormValue("order_status")

    err = h.Repo.Order.UpdateStatus(id, OrderStatus(status))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    err = h.Tmpl.ExecuteTemplate(w, "orderTable", nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

/*** End User actions ***/

/*** utils ***/
func makeRange(min int, max int) []int {
    rangeArray := make([]int, max-min+1)
    for i := range rangeArray {
        rangeArray[i] = min + i
    }
    return rangeArray
}

type ProductMessage struct {
    Messages []string
    Product  *Product
}

func sendMessage(w http.ResponseWriter, tmpl *template.Template, messages []string, product *Product) {
    data := ProductMessage{Messages: messages, Product: product}
    err := tmpl.ExecuteTemplate(w, "viewMessages", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

type Cart struct {
    Items       []OrderItem
    Message     string
    AlertType   string
    TotalCost   float64
    RefreshCart bool
}

func getCartTotal() float64 {
    totalCost := 0.0
    for _, item := range cartItems {
        totalCost += item.Cost
    }
    return totalCost
}

func getCartItem(id uuid.UUID) (int, *OrderItem) {
    index := slices.IndexFunc(cartItems, func(item OrderItem) bool {
        return item.ProductID == id
    })
    return index, &cartItems[index]
}
