package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/Aakashc137/gomsa-proto/user"
)

// server struct implements the UserServiceServer interface
type server struct {
	user.UnimplementedUserServiceServer
	db *pgxpool.Pool
}

// CreateUser implements the CreateUser RPC method.
func (s *server) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.CreateUserResponse, error) {
	// Insert user into the database and retrieve the generated ID.
	var id uint64
	err := s.db.QueryRow(ctx, `
        INSERT INTO users (name, email, created_at, updated_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `,
		req.GetName(),
		req.GetEmail(),
		time.Now(),
		time.Now(),
	).Scan(&id)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		return nil, grpc.Errorf(grpc.Code(err), "Failed to create user")
	}

	return &user.CreateUserResponse{
		User: &user.User{
			Id:        id,
			Name:      req.GetName(),
			Email:     req.GetEmail(),
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}, nil
}

// GetUser implements the GetUser RPC method.
func (s *server) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
	var u user.User
	err := s.db.QueryRow(ctx, `
        SELECT id, name, email, created_at, updated_at
        FROM users
        WHERE id = $1
    `, req.GetId()).Scan(
		&u.Id,
		&u.Name,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return nil, grpc.Errorf(grpc.Code(err), "Failed to get user")
	}

	return &user.GetUserResponse{
		User: &u,
	}, nil
}

// UpdateUser implements the UpdateUser RPC method.
func (s *server) UpdateUser(ctx context.Context, req *user.UpdateUserRequest) (*user.UpdateUserResponse, error) {
	var u user.User
	err := s.db.QueryRow(ctx, `
        UPDATE users
        SET name = $1, email = $2, updated_at = $3
        WHERE id = $4
        RETURNING id, name, email, created_at, updated_at
    `, req.GetName(), req.GetEmail(), time.Now(), req.GetId()).Scan(
		&u.Id,
		&u.Name,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		log.Printf("Failed to update user: %v", err)
		return nil, grpc.Errorf(grpc.Code(err), "Failed to update user")
	}

	return &user.UpdateUserResponse{
		User: &u,
	}, nil
}

// DeleteUser implements the DeleteUser RPC method.
func (s *server) DeleteUser(ctx context.Context, req *user.DeleteUserRequest) (*user.DeleteUserResponse, error) {
	_, err := s.db.Exec(ctx, `
        DELETE FROM users
        WHERE id = $1
    `, req.GetId())
	if err != nil {
		log.Printf("Failed to delete user: %v", err)
		return nil, grpc.Errorf(grpc.Code(err), "Failed to delete user")
	}

	return &user.DeleteUserResponse{
		Message: "User deleted successfully",
	}, nil
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Proceeding with environment variables.")
	}

	// Retrieve DATABASE_URL from environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Connect to PostgreSQL
	dbpool, err := pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbpool.Close()

	// Listen on port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a new gRPC server
	grpcServer := grpc.NewServer()

	// Register the UserService server
	user.RegisterUserServiceServer(grpcServer, &server{db: dbpool})

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	// Graceful shutdown handling
	go func() {
		// Wait for interrupt signal to gracefully shutdown the server
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		log.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Println("gRPC server is running on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
