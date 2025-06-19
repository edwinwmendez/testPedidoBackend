package main

import (
	"fmt"
	"log"
	"os"

	"backend/config"
	"backend/database"
	"backend/internal/models"
)

func main() {
	fmt.Println("🗑️  Iniciando limpieza de pedidos...")

	// Cargar configuración
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error cargando configuración: %v", err)
	}

	// Conectar a la base de datos
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}

	// Confirmar antes de proceder
	fmt.Print("⚠️  ADVERTENCIA: Esto eliminará TODOS los pedidos y sus items.\n¿Estás seguro? (escriba 'SI' para confirmar): ")
	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "SI" {
		fmt.Println("❌ Operación cancelada.")
		os.Exit(0)
	}

	// Contar registros antes de eliminar
	var orderCount, orderItemCount int64
	db.Model(&models.Order{}).Count(&orderCount)
	db.Model(&models.OrderItem{}).Count(&orderItemCount)

	fmt.Printf("📊 Registros encontrados:\n")
	fmt.Printf("   - Pedidos: %d\n", orderCount)
	fmt.Printf("   - Items de pedidos: %d\n", orderItemCount)

	if orderCount == 0 && orderItemCount == 0 {
		fmt.Println("✅ No hay registros para eliminar.")
		return
	}

	// Usar una transacción para seguridad
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Fatalf("❌ Error durante la eliminación: %v", r)
		}
	}()

	fmt.Println("🗑️  Eliminando items de pedidos...")
	result := tx.Where("1 = 1").Delete(&models.OrderItem{})
	if result.Error != nil {
		tx.Rollback()
		log.Fatalf("❌ Error eliminando items de pedidos: %v", result.Error)
	}
	fmt.Printf("✅ %d items de pedidos eliminados.\n", result.RowsAffected)

	fmt.Println("🗑️  Eliminando pedidos...")
	result = tx.Where("1 = 1").Delete(&models.Order{})
	if result.Error != nil {
		tx.Rollback()
		log.Fatalf("❌ Error eliminando pedidos: %v", result.Error)
	}
	fmt.Printf("✅ %d pedidos eliminados.\n", result.RowsAffected)

	// Confirmar la transacción
	if err := tx.Commit().Error; err != nil {
		log.Fatalf("❌ Error confirmando la transacción: %v", err)
	}

	// Verificar que las tablas estén vacías
	var finalOrderCount, finalOrderItemCount int64
	db.Model(&models.Order{}).Count(&finalOrderCount)
	db.Model(&models.OrderItem{}).Count(&finalOrderItemCount)

	fmt.Printf("\n📊 Conteo final:\n")
	fmt.Printf("   - Pedidos: %d\n", finalOrderCount)
	fmt.Printf("   - Items de pedidos: %d\n", finalOrderItemCount)

	if finalOrderCount == 0 && finalOrderItemCount == 0 {
		fmt.Println("\n🎉 ¡Limpieza completada exitosamente!")
	} else {
		fmt.Printf("\n⚠️  Advertencia: Aún quedan registros en las tablas.\n")
	}
}
