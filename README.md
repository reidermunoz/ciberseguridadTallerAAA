# AutoMarket · Backend con AAA

API HTTP en Go (librería estándar + Gin) para el catálogo de vehículos usados,
con Authentication, Authorization y Accounting sobre todas sus operaciones.

## Ejecución

```bash
export AUTOMARKET_AUDIT_KEY="una-clave-secreta-de-al-menos-32-caracteres"
export AUTOMARKET_ADMIN_EMAIL="admin@automarket.co"
export AUTOMARKET_ADMIN_PASSWORD="AdminSegura123"
go run .
```

| Variable | Obligatoria | Uso |
|---|---|---|
| `AUTOMARKET_AUDIT_KEY` | sí | Clave HMAC que sella la bitácora (mín. 32 caracteres) |
| `AUTOMARKET_ADMIN_EMAIL` / `AUTOMARKET_ADMIN_PASSWORD` | sí | Cuenta de administrador creada al arrancar si no existe |
| `AUTOMARKET_ADMIN_NAME` | no | Nombre del administrador (`Administrador`) |
| `AUTOMARKET_ADDR` | no | Dirección de escucha (`localhost:8080`) |
| `AUTOMARKET_DATA_DIR` | no | Carpeta de persistencia (`data`) |

## Operaciones

| Método | Ruta | Rol | Historia |
|---|---|---|---|
| GET | `/vehicles` | público | HU-VIS-01 |
| GET | `/vehicles/:id` | público | HU-VIS-01 |
| POST | `/auth/register` | público | HU-VEN-01 |
| POST | `/auth/login` | público | HU-VEN-01 / HU-ADM-01 |
| POST | `/vehicles` | vendedor | HU-VEN-01 |
| PATCH | `/vehicles/:id/sold` | vendedor dueño | HU-VEN-01 |
| PATCH | `/vehicles/:id/approve` | administrador | HU-ADM-01 |
| DELETE | `/vehicles/:id` | administrador | HU-ADM-01 |
| DELETE | `/users/:id` | administrador | HU-ADM-01 |

Las rutas protegidas requieren `Authorization: Bearer <token>` con el token
devuelto por `/auth/login`.

## AAA

- **Authentication** (`middleware/authentication.go`): token aleatorio de 128 bits
  (`crypto/rand`) con expiración de 1 hora; contraseñas derivadas con PBKDF2-SHA256
  (600 000 iteraciones, sal aleatoria) y comparadas en tiempo constante.
- **Authorization** (`middleware/authorization.go`): control por rol en cada grupo
  de rutas; la regla de propiedad (solo el dueño reporta su venta) está en `service`.
- **Accounting** (`middleware/accounting.go`, `audit/`): toda solicitud, incluidas
  las públicas y las rechazadas, se agrega a `data/audit.log`. Cada entrada guarda
  el hash de la anterior y un HMAC-SHA256; al arrancar se verifica la cadena y, si
  fue alterada, el servidor no inicia.

## Estructura

```
main.go, router.go   configuración, arranque y rutas
middleware/          Authentication, Authorization, Accounting
handlers/            HTTP ⇄ servicios, validación y mapeo de errores
service/             reglas de negocio de las HU
auth/                contraseñas, sesiones e identidad del solicitante
audit/               bitácora encadenada con HMAC
storage/             tablas persistidas en archivos JSON
models/              entidades del dominio
```
