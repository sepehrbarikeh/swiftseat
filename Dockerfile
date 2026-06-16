# # مرحله اول: بیلد
# FROM golang:alpine AS builder
# WORKDIR /app
# COPY go.mod go.sum ./
# RUN go mod download
# COPY . .
# # باینری رو می‌سازیم
# RUN go build -o main ./cmd/main.go

# # مرحله دوم: اجرا
# FROM alpine:latest
# WORKDIR /app
# # کپی کردن باینری و فایل کانفیگ در یک مسیر واحد
# COPY --from=builder /app/main .
# COPY --from=builder /app/config.yml . 

# EXPOSE 8080
# CMD ["./main"]