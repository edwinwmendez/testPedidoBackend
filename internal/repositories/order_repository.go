package repositories

import (
	"math"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(order *models.Order) error
	FindByID(id string) (*models.Order, error)
	FindAll() ([]*models.Order, error)
	FindByClientID(clientID string) ([]*models.Order, error)
	FindByRepartidorID(repartidorID string) ([]*models.Order, error)
	FindByStatus(status models.OrderStatus) ([]*models.Order, error)
	FindPendingOrders() ([]*models.Order, error)
	FindNearbyOrders(lat, lng float64, radiusKm float64) ([]*models.Order, error)
	Update(order *models.Order) error
	UpdateStatus(id string, status models.OrderStatus) error
	AssignRepartidor(orderID string, repartidorID string) error
	SetEstimatedArrivalTime(orderID string, eta time.Time) error
	Delete(id string) error
	AddOrderItem(item *models.OrderItem) error
	FindOrderItems(orderID string) ([]*models.OrderItem, error)
	DeleteOrderItem(itemID string) error
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{
		db: db,
	}
}

func (r *orderRepository) Create(order *models.Order) error {
	return r.db.Create(order).Error
}

func (r *orderRepository) FindByID(id string) (*models.Order, error) {
	var order models.Order
	err := r.db.
		Preload("Client").
		Preload("AssignedRepartidor").
		Preload("OrderItems.Product"). // Preload anidado para productos
		Where("order_id = ?", id).
		First(&order).Error

	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderRepository) FindAll() ([]*models.Order, error) {
	var orders []*models.Order

	if err := r.db.Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) FindByClientID(clientID string) ([]*models.Order, error) {
	var orders []*models.Order

	if err := r.db.Where("client_id = ?", clientID).Order("order_time DESC").Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) FindByRepartidorID(repartidorID string) ([]*models.Order, error) {
	var orders []*models.Order

	if err := r.db.Where("assigned_repartidor_id = ?", repartidorID).Order("order_time DESC").Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) FindByStatus(status models.OrderStatus) ([]*models.Order, error) {
	var orders []*models.Order

	if err := r.db.Where("order_status = ?", status).Order("order_time ASC").Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) FindPendingOrders() ([]*models.Order, error) {
	var orders []*models.Order

	if err := r.db.Where("order_status IN ?", []models.OrderStatus{models.OrderStatusPending, models.OrderStatusPendingOutOfHours}).
		Order("order_time ASC").Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) FindNearbyOrders(lat, lng float64, radiusKm float64) ([]*models.Order, error) {
	// Implementación básica de búsqueda por distancia euclidiana
	// Para una implementación más precisa, se recomienda usar PostGIS en producción
	var orders []*models.Order

	// Aproximación de conversión de grados a km (1 grado ≈ 111.32 km en el ecuador)
	// Esta es una aproximación simple para el MVP
	latDiff := radiusKm / 111.32
	lngDiff := radiusKm / (111.32 * cos(lat*(3.14159/180.0)))

	if err := r.db.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
		lat-latDiff, lat+latDiff, lng-lngDiff, lng+lngDiff).
		Where("order_status = ?", models.OrderStatusPending).
		Find(&orders).Error; err != nil {
		return nil, err
	}

	// Filtrar más precisamente calculando la distancia real
	var result []*models.Order
	for _, order := range orders {
		distance := calculateDistance(lat, lng, order.Latitude, order.Longitude)
		if distance <= radiusKm {
			result = append(result, order)
		}
	}

	return result, nil
}

func (r *orderRepository) Update(order *models.Order) error {
	return r.db.Save(order).Error
}

func (r *orderRepository) UpdateStatus(id string, status models.OrderStatus) error {
	updates := map[string]interface{}{"order_status": status}

	// Actualizar campos adicionales según el nuevo estado
	now := time.Now()
	switch status {
	case models.OrderStatusConfirmed:
		updates["confirmed_at"] = now
	case models.OrderStatusAssigned:
		updates["assigned_at"] = now
	case models.OrderStatusDelivered:
		updates["delivered_at"] = now
	case models.OrderStatusCancelled:
		updates["cancelled_at"] = now
	}

	return r.db.Model(&models.Order{}).Where("order_id = ?", id).Updates(updates).Error
}

func (r *orderRepository) AssignRepartidor(orderID string, repartidorID string) error {
	updates := map[string]interface{}{
		"assigned_repartidor_id": repartidorID,
		"assigned_at":            time.Now(),
	}

	return r.db.Model(&models.Order{}).Where("order_id = ?", orderID).Updates(updates).Error
}

func (r *orderRepository) SetEstimatedArrivalTime(orderID string, eta time.Time) error {
	return r.db.Model(&models.Order{}).Where("order_id = ?", orderID).
		Update("estimated_arrival_time", eta).Error
}

func (r *orderRepository) Delete(id string) error {
	// Primero eliminar los items relacionados
	if err := r.db.Where("order_id = ?", id).Delete(&models.OrderItem{}).Error; err != nil {
		return err
	}

	// Luego eliminar la orden
	return r.db.Delete(&models.Order{}, "order_id = ?", id).Error
}

func (r *orderRepository) AddOrderItem(item *models.OrderItem) error {
	return r.db.Create(item).Error
}

func (r *orderRepository) FindOrderItems(orderID string) ([]*models.OrderItem, error) {
	var items []*models.OrderItem

	if err := r.db.Where("order_id = ?", orderID).Preload("Product").Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *orderRepository) DeleteOrderItem(itemID string) error {
	return r.db.Delete(&models.OrderItem{}, "order_item_id = ?", itemID).Error
}

// Funciones auxiliares para cálculos geoespaciales
func cos(x float64) float64 {
	return float64(math.Cos(float64(x)))
}

func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Implementación de la fórmula de Haversine para calcular distancia entre dos puntos geográficos
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0

	lat1 = lat1 * math.Pi / 180.0
	lat2 = lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
