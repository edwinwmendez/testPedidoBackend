# Admin User Management Endpoints Implementation

## Summary
Successfully implemented comprehensive admin user management endpoints in the AppGas backend API.

## Implemented Endpoints

### 1. GET /api/v1/admin/users
- **Description**: List all users with pagination and filtering
- **Features**:
  - Pagination support (page, page_size parameters)
  - Role filtering (CLIENT, REPARTIDOR, ADMIN)
  - Returns total count and pagination metadata
- **Authorization**: Admin only
- **Response**: `PaginatedUsersResponse` with users array and pagination info

### 2. POST /api/v1/admin/users
- **Description**: Create a new user
- **Features**:
  - Creates users with all roles
  - Validates required fields (email, password, full_name, phone_number, user_role)
  - Handles password hashing
  - Prevents duplicate emails
- **Authorization**: Admin only
- **Request**: `CreateUserRequest`
- **Response**: Created `User` object

### 3. PUT /api/v1/admin/users/{id}
- **Description**: Update an existing user
- **Features**:
  - Partial updates (only provided fields are updated)
  - Supports updating email, full_name, phone_number, and user_role
  - Role validation
- **Authorization**: Admin only
- **Request**: `UpdateUserAdminRequest`
- **Response**: Updated `User` object

### 4. PUT /api/v1/admin/users/{id}/activate
- **Description**: Activate a user account
- **Features**:
  - Sets user's is_active field to true
  - Admin action logging
- **Authorization**: Admin only
- **Response**: Success message

### 5. PUT /api/v1/admin/users/{id}/deactivate
- **Description**: Deactivate a user account
- **Features**:
  - Sets user's is_active field to false
  - Admin action logging
- **Authorization**: Admin only
- **Response**: Success message

### 6. DELETE /api/v1/admin/users/{id}
- **Description**: Permanently delete a user
- **Features**:
  - Hard delete from database
  - Validates user exists before deletion
  - Admin action logging
- **Authorization**: Admin only
- **Response**: Success message

## Technical Implementation

### Repository Layer (`internal/repositories/user_repository.go`)
- Added `FindAll()` method for retrieving all users
- Added `FindAllWithPagination()` method with role filtering and pagination
- Added `SoftDelete()` method for future soft delete functionality

### Service Layer (`internal/services/user_service.go`)
- Added `PaginatedUsersResponse` struct for paginated responses
- Added `GetUsersWithPagination()` method with validation
- Added admin-specific methods with logging:
  - `CreateUserAdmin()`
  - `UpdateUserAdmin()`
  - `ActivateUser()`
  - `DeactivateUser()`
  - `DeleteUserAdmin()`

### Handler Layer (`api/v1/handlers/user_handler.go`)
- Added request/response structs:
  - `CreateUserRequest`
  - `UpdateUserAdminRequest`
- Added admin handler methods with proper validation and error handling:
  - `GetAllUsersAdmin()`
  - `CreateUserAdmin()`
  - `UpdateUserAdmin()`
  - `ActivateUserAdmin()`
  - `DeactivateUserAdmin()`
  - `DeleteUserAdmin()`

### Routing (`api/v1/handlers/user_handler.go`)
- Added admin routes under `/api/v1/admin/users`
- Applied proper authentication and authorization middleware
- All admin endpoints require admin role

## Security Features
- **Authentication**: All endpoints require valid JWT token
- **Authorization**: All endpoints require admin role
- **Logging**: All admin actions are logged with admin ID
- **Validation**: Input validation for all request parameters
- **Error Handling**: Comprehensive error responses

## Error Handling
- 400 Bad Request: Invalid input data, validation errors
- 401 Unauthorized: Missing or invalid authentication
- 403 Forbidden: Insufficient permissions (non-admin users)
- 404 Not Found: User not found
- 409 Conflict: Email already exists (create user)
- 500 Internal Server Error: Server-side errors

## Pagination
- Default page size: 10
- Maximum page size: 100
- Returns total count, current page, page size, and total pages
- Results ordered by creation date (newest first)

## Logging
All admin actions are logged with the following format:
- Admin ID performing the action
- Action being performed
- Target user ID/email
- Success/failure status

## Testing
- All existing tests continue to pass
- Integration with existing authentication and authorization middleware
- Follows existing code patterns and conventions

## Example Usage

### List Users with Pagination
```bash
GET /api/v1/admin/users?page=1&page_size=10&role=CLIENT
Authorization: Bearer <admin-jwt-token>
```

### Create New User
```bash
POST /api/v1/admin/users
Authorization: Bearer <admin-jwt-token>
Content-Type: application/json

{
  "email": "newuser@example.com",
  "password": "secure123",
  "full_name": "New User",
  "phone_number": "+1234567890",
  "user_role": "CLIENT"
}
```

### Update User
```bash
PUT /api/v1/admin/users/123e4567-e89b-12d3-a456-426614174000
Authorization: Bearer <admin-jwt-token>
Content-Type: application/json

{
  "full_name": "Updated Name",
  "user_role": "REPARTIDOR"
}
```

### Activate/Deactivate User
```bash
PUT /api/v1/admin/users/123e4567-e89b-12d3-a456-426614174000/activate
Authorization: Bearer <admin-jwt-token>
```

All endpoints follow the existing API patterns and are fully integrated with the current authentication and authorization system.