# Elterngeld Portal API

Ein vollständig strukturiertes, produktionsreifes Webserver-Projekt in Go für ein Elterngeldlotsen-Portal. Die REST-API unterstützt drei Benutzerrollen: Kunden (User), Berater und Admins mit umfassender Infrastruktur für Zahlungen, Authentifizierung, Datei-Uploads, Workflow-Engine und mehr.

## 🚀 Features

### 🏗️ Architektur
- **Clean Architecture** mit Trennung von Handler, Service, Repository und Middleware
- **Go Module-basiertes Setup** mit allen notwendigen Dependencies
- **Strukturierte Aufteilung** in `/cmd`, `/internal`, `/pkg`, `/config`
- **Modularer Wechsel** zwischen SQLite (Dev) und PostgreSQL (Prod)

### 🔐 Authentifizierung & Autorisierung
- **JWT-basiertes Auth-System** mit Access- & Refresh-Token-Flow
- **Passwort-Hashing** mit bcrypt
- **Refresh Token Rotation**
- **Role-Based Access Control** (User, Berater, Admin)
- **Token Blacklisting** für sicheres Logout

### 💾 Datenbank & Migration
- **GORM** als ORM mit PostgreSQL/SQLite Support
- **Automatische Migrations** mit `golang-migrate`
- **Seed-Daten** für Entwicklung
- **Connection Pooling** und Health Checks

### 📁 File Management
- **Multipart File Upload** mit Validierung
- **Lokale Speicherung** oder **S3-Integration**
- **MIME-Type Validierung** (PDF, PNG, JPG)
- **Upload-Historie** pro Lead

### 💳 Zahlungsabwicklung
- **Stripe Integration** für Online-Zahlungen
- **Webhook-Handler** für Zahlungsbestätigungen
- **Rechnungsmanagement**
- **Refund-Funktionalität**

### 🔄 Workflow Engine
- **Lead-Status Management** mit definierten Übergängen
- **Automatisches Activity Logging**
- **E-Mail-Benachrichtigungen** bei Statusänderungen

### ✉️ E-Mail System
- **SMTP** oder **Mailgun** Support
- **Template-basierte E-Mails**
- **Event-gesteuerte Benachrichtigungen**

### 🛡️ Middleware & Sicherheit
- **Auth Middleware** mit JWT-Validierung
- **Role-based Access Control**
- **Request Logging** mit Zap
- **Rate Limiting**
- **CORS-Unterstützung**
- **Security Headers**

### 📊 Monitoring & Observability
- **Structured Logging** mit Zap
- **Activity Logging** in Datenbank
- **Health Check Endpoints**
- **Request ID Tracking**

### 📚 Dokumentation
- **Swagger/OpenAPI** Dokumentation
- **API-Dokumentation** unter `/docs`
- **Postman Collection** (geplant)

## 🛠️ Technologie-Stack

- **Backend**: Go 1.22+
- **Web Framework**: Gin
- **ORM**: GORM
- **Datenbank**: PostgreSQL / SQLite
- **Authentication**: JWT
- **Payments**: Stripe
- **File Storage**: Local FS / AWS S3
- **Email**: SMTP / Mailgun
- **Logging**: Zap
- **Documentation**: Swagger
- **Containerization**: Docker & Docker Compose
- **Testing**: Go testing package

## 📋 Voraussetzungen

- Go 1.22 oder höher
- Docker & Docker Compose (optional)
- Make (für Automatisierung)
- Git

## 🚀 Quick Start

### 1. Repository klonen
```bash
git clone <repository-url>
cd elterngeld-portal
```

### 2. Projekt Setup
```bash
make setup
```
Dieser Befehl:
- Installiert alle Dependencies
- Installiert Entwicklungstools
- Erstellt .env-Datei
- Erstellt notwendige Verzeichnisse

### 3. Konfiguration anpassen
```bash
# .env-Datei bearbeiten
cp .env.example .env
nano .env
```

### 4. Anwendung starten

#### Lokale Entwicklung (SQLite)
```bash
make dev
```

#### Mit Docker Compose (PostgreSQL + Services)
```bash
docker-compose up -d
```

### 5. API testen
```bash
# Health Check
curl http://localhost:8080/health

# API Dokumentation
open http://localhost:8080/docs/index.html
```

## 📖 Makefile Kommandos

```bash
# Hilfe anzeigen
make help

# Entwicklung
make dev          # Dependencies installieren und Server starten
make run          # Server starten
make watch        # Auto-reload mit air

# Build
make build        # Anwendung builden
make build-all    # Für alle Plattformen builden

# Testing
make test         # Tests ausführen
make test-coverage # Tests mit Coverage
make test-race    # Race-Detection Tests

# Datenbank
make migrate      # Migrationen ausführen
make seed         # Testdaten einfügen
make db-reset     # Datenbank zurücksetzen

# Code Quality
make fmt          # Code formatieren
make lint         # Linter ausführen
make check        # Alle Quality Checks

# Docker
make docker-up    # Services mit Docker Compose starten
make docker-down  # Services stoppen
make docker-logs  # Logs anzeigen

# Dokumentation
make swagger      # Swagger-Docs generieren
```

## 🗂️ Projektstruktur

```
elterngeld-portal/
├── cmd/
│   └── server/           # Main application
├── internal/
│   ├── database/         # Database connection & migrations
│   ├── middleware/       # HTTP middleware
│   ├── models/          # Data models
│   └── server/          # HTTP server setup
├── pkg/
│   ├── auth/            # Authentication logic
│   └── logger/          # Logging utilities
├── config/              # Configuration management
├── storage/             # File uploads
├── data/               # SQLite database
├── docs/               # Swagger documentation
├── migrations/         # Database migrations
├── tests/              # Test files
├── .env.example        # Environment template
├── docker-compose.yml  # Docker services
├── Dockerfile          # Container definition
├── Makefile           # Automation commands
└── README.md          # This file
```

## 📊 API Endpunkte

### 🔐 Authentifizierung
```
POST /api/v1/auth/register      # Benutzer registrieren
POST /api/v1/auth/login         # Anmelden
POST /api/v1/auth/refresh       # Token erneuern
POST /api/v1/auth/logout        # Abmelden
GET  /api/v1/auth/me           # Aktueller Benutzer
```

### 👥 Benutzer
```
GET    /api/v1/users           # Benutzer auflisten (Berater/Admin)
GET    /api/v1/users/:id       # Benutzer anzeigen
PUT    /api/v1/users/:id       # Benutzer aktualisieren
DELETE /api/v1/users/:id       # Benutzer löschen (Admin)
```

### 📋 Leads
```
GET    /api/v1/leads           # Leads auflisten
POST   /api/v1/leads           # Lead erstellen
GET    /api/v1/leads/:id       # Lead anzeigen
PUT    /api/v1/leads/:id       # Lead aktualisieren
PATCH  /api/v1/leads/:id/status # Status ändern
POST   /api/v1/leads/:id/assign # Lead zuweisen
```

### 📄 Dokumente
```
GET    /api/v1/documents       # Dokumente auflisten
POST   /api/v1/documents       # Dokument hochladen
GET    /api/v1/documents/:id   # Dokument anzeigen
DELETE /api/v1/documents/:id   # Dokument löschen
GET    /api/v1/documents/:id/download # Dokument herunterladen
```

### 💳 Zahlungen
```
GET    /api/v1/payments        # Zahlungen auflisten
POST   /api/v1/payments/checkout # Stripe Checkout erstellen
GET    /api/v1/payments/:id    # Zahlung anzeigen
POST   /api/v1/payments/:id/refund # Rückerstattung
```

### 📈 Admin
```
GET    /api/v1/admin/stats     # Admin-Statistiken
GET    /api/v1/admin/users     # Alle Benutzer
POST   /api/v1/admin/users     # Benutzer erstellen
PUT    /api/v1/admin/users/:id/role # Rolle ändern
```

## 🌐 Benutzerrollen

### 👤 User (Kunde)
- Eigene Leads erstellen und verwalten
- Dokumente hochladen
- Zahlungen durchführen
- Eigene Daten einsehen und aktualisieren

### 👨‍💼 Berater
- Zugewiesene Leads bearbeiten
- Status von Leads ändern
- Kommentare hinzufügen
- Kundendokumente einsehen

### 👑 Admin
- Alle Systemfunktionen
- Benutzer verwalten
- Leads zuweisen
- System-Statistiken einsehen
- Zahlungen verwalten

## ⚙️ Konfiguration

Die Anwendung verwendet eine `.env`-Datei für die Konfiguration:

```env
# Server
PORT=8080
ENV=development

# Datenbank
DB_DRIVER=sqlite
DB_HOST=localhost
SQLITE_PATH=./data/database.db

# JWT
JWT_SECRET=your-secret-key
JWT_ACCESS_EXPIRY=15m

# Stripe
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# E-Mail
EMAIL_PROVIDER=smtp
SMTP_HOST=localhost
SMTP_PORT=1025
```

## 🧪 Testing

```bash
# Alle Tests ausführen
make test

# Tests mit Coverage
make test-coverage

# Integration Tests
make test-integration

# Race Condition Tests
make test-race
```

## 🚀 Deployment

### Docker Deployment
```bash
# Image bauen
make docker-build

# Mit Docker Compose
docker-compose up -d

# Produktions-Build
make prod-build
```

### Manuelle Deployment
```bash
# Für Linux bauen
make build-linux

# Binary kopieren und ausführen
./build/elterngeld-portal-linux
```

## 📝 Entwicklung

### Neue Migration erstellen
```bash
make migrate-create name=add_new_table
```

### Swagger-Dokumentation aktualisieren
```bash
make swagger
```

### Code formatieren
```bash
make fmt
```

### Linting
```bash
make lint
```

## 🔧 Troubleshooting

### Häufige Probleme

1. **Port bereits belegt**
   ```bash
   # Port in .env ändern oder Prozess beenden
   lsof -ti:8080 | xargs kill
   ```

2. **Datenbankverbindung fehlgeschlagen**
   ```bash
   # Datenbank-Service prüfen
   make docker-ps
   ```

3. **Dependencies fehlen**
   ```bash
   # Dependencies neu installieren
   make deps
   ```

## 📈 Performance

- **SQLite**: Gut für Entwicklung und kleine Deployments
- **PostgreSQL**: Empfohlen für Produktion
- **Connection Pooling**: Automatisch konfiguriert
- **Rate Limiting**: Schutz vor Abuse
- **Caching**: Ready für Redis-Integration

## 🔒 Sicherheit

- **JWT mit sicheren Secrets**
- **Passwort-Hashing mit bcrypt**
- **SQL Injection Schutz durch GORM**
- **XSS Protection Headers**
- **CORS-Konfiguration**
- **File Upload Validierung**
- **Rate Limiting**

## 🤝 Contributing

1. Fork das Repository
2. Feature Branch erstellen (`git checkout -b feature/AmazingFeature`)
3. Änderungen committen (`git commit -m 'Add some AmazingFeature'`)
4. Branch pushen (`git push origin feature/AmazingFeature`)
5. Pull Request erstellen

## 📄 Lizenz

Dieses Projekt steht unter der MIT-Lizenz. Siehe [LICENSE](LICENSE) für Details.

## 👥 Support

Bei Fragen oder Problemen:
- GitHub Issues erstellen
- E-Mail: support@elterngeld-portal.de
- Dokumentation: `http://localhost:8080/docs`

## 🗺️ Roadmap

- [ ] Frontend (React/Vue.js)
- [ ] Mobile App (React Native/Flutter)
- [ ] PDF-Generierung für Anträge
- [ ] Erweiterte Reporting-Features
- [ ] Multi-Tenant Architektur
- [ ] GraphQL API
- [ ] Real-time Notifications

## 🙏 Acknowledgments

- [Gin Web Framework](https://gin-gonic.com/)
- [GORM ORM](https://gorm.io/)
- [Zap Logging](https://github.com/uber-go/zap)
- [Stripe](https://stripe.com/)
- [Swagger](https://swagger.io/)