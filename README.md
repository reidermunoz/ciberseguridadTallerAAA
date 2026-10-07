# AutoMarket · Backend con AAA

API HTTP en Go (librería estándar + Gin) para el catálogo de vehículos usados,
con Authentication, Authorization y Accounting sobre todas sus operaciones.

> ¿Quieres comprobar que funciona? Sigue la
> [guía de pruebas paso a paso](#guía-de-pruebas-paso-a-paso) al final de este
> documento: va desde clonar el repositorio hasta entregar el Pull Request en GitHub.

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

---

# Guía de pruebas paso a paso

Esta guía recorre las tres historias de usuario (HU-VIS-01, HU-VEN-01 y
HU-ADM-01) y las tres A del RF-01, y explica qué respuesta debe aparecer en cada
paso. Termina con el flujo de entrega en GitHub.

## Paso 0 · Requisitos

| Herramienta | Para qué | Cómo comprobarla |
|---|---|---|
| Go 1.27 o superior | Compilar y ejecutar el backend | `go version` |
| Git | Clonar y subir cambios | `git --version` |
| Git Bash (viene con Git en Windows) | Ejecutar los comandos `curl` de esta guía | Abrir "Git Bash" desde el menú Inicio |

> **Importante en Windows:** usa **Git Bash**, no PowerShell. En PowerShell 5.1
> `curl` es un alias de otro comando y las comillas del JSON se dañan.
> Además, escribe los datos de prueba **sin tildes** (la consola de Windows altera
> los acentos al enviarlos; el servidor los maneja bien).

## Paso 1 · Obtener el código

```bash
git clone https://github.com/reidermunoz/ciberseguridadTallerAAA.git
cd ciberseguridadTallerAAA
git checkout feature/aaa     # el código vive en esta rama hasta que se acepte el PR
```

Si ya tienes la carpeta del proyecto, basta con entrar a ella con `cd`.

## Paso 2 · Arrancar el servidor (terminal 1)

Abre una ventana de Git Bash en la carpeta del proyecto y ejecuta:

```bash
rm -rf data      # empieza con datos limpios: los IDs de esta guía asumen eso

export AUTOMARKET_AUDIT_KEY="clave-secreta-de-auditoria-de-32-chars"
export AUTOMARKET_ADMIN_EMAIL="admin@automarket.co"
export AUTOMARKET_ADMIN_PASSWORD="AdminSegura123"

go run .
```

**Esperado:** Gin lista las 9 rutas y al final aparece

```
AutoMarket escuchando en http://localhost:8080
[GIN-debug] Listening and serving HTTP on localhost:8080
```

Deja esta ventana abierta: es el servidor. Al arrancar se crea la carpeta `data/`
y el administrador queda registrado con el **id 1**.

> Si falta alguna variable el servidor no arranca y lo dice, por ejemplo:
> `AUTOMARKET_AUDIT_KEY debe tener al menos 32 caracteres`.

## Paso 3 · Preparar la terminal de pruebas (terminal 2)

Abre **otra** ventana de Git Bash en la carpeta del proyecto y define dos atajos:

```bash
B=http://localhost:8080
J='Content-Type: application/json'
```

En todos los comandos se usa `curl -i`, que muestra la línea `HTTP/1.1 <código>`.
Ese código es la evidencia de que cada control funciona.

## Paso 4 · HU-VEN-01 · Registro e inicio de sesión del vendedor

**4.1 Registrar a la vendedora Ana**

```bash
curl -i -X POST $B/auth/register -H "$J" \
  -d '{"name":"Ana Perez","email":"ana@mail.com","phone":"3001234567","password":"Vendedora123"}'
```

Esperado: `201 Created` y `{"email":"ana@mail.com","id":2,"role":"vendedor"}`.
Todo registro crea un **vendedor**; no hay forma de registrarse como administrador.

**4.2 Intentar registrar el mismo correo**

```bash
curl -i -X POST $B/auth/register -H "$J" \
  -d '{"name":"Ana","email":"ana@mail.com","phone":"1","password":"Vendedora123"}'
```

Esperado: `409 Conflict` · `el correo ya está registrado`.

**4.3 Datos inválidos**

```bash
curl -i -X POST $B/auth/register -H "$J" -d '{"email":"no-es-correo","password":"123"}'
```

Esperado: `400 Bad Request` con el detalle de cada campo inválido.

**4.4 Login con contraseña incorrecta**

```bash
curl -i -X POST $B/auth/login -H "$J" -d '{"email":"ana@mail.com","password":"mala"}'
```

Esperado: `401 Unauthorized` · `credenciales inválidas`.

**4.5 Login correcto y guardar el token**

```bash
TS=$(curl -s -X POST $B/auth/login -H "$J" \
  -d '{"email":"ana@mail.com","password":"Vendedora123"}' | sed 's/.*"token":"\([^"]*\)".*/\1/')
echo $TS
```

Esperado: se imprime un token de 26 caracteres. La variable `TS` guarda el token
de la vendedora.

## Paso 5 · RF-01 Authentication · sin identidad no hay operación protegida

**5.1 Publicar sin token**

```bash
curl -i -X POST $B/vehicles -H "$J" \
  -d '{"brand":"Renault","model":"Koleos","year":2018,"price":78000000,"mileage":99000}'
```

Esperado: `401 Unauthorized` · `se requiere autenticación`.

**5.2 Publicar con un token inventado**

```bash
curl -i -X POST $B/vehicles -H "Authorization: Bearer inventado" -H "$J" -d '{}'
```

Esperado: `401 Unauthorized` · `token inválido o expirado`.

## Paso 6 · HU-VEN-01 · Publicar un vehículo (queda pendiente)

```bash
curl -i -X POST $B/vehicles -H "Authorization: Bearer $TS" -H "$J" \
  -d '{"brand":"Renault","model":"Koleos","year":2018,"price":78000000,"mileage":99000}'
```

Esperado: `201 Created`, `"id":1`, `"seller_id":2` y **`"status":"pendiente"`**.
Así se cumple el criterio de aceptación de HU-VEN-01: la publicación queda
pendiente de aprobación y asociada a la cuenta de la vendedora.

**El visitante todavía no la ve:**

```bash
curl -i $B/vehicles        # 200 y []
curl -i $B/vehicles/1      # 404 · recurso no encontrado
```

## Paso 7 · RF-01 Authorization · cada rol solo hace lo suyo

**7.1 La vendedora intenta aprobar su propia publicación**

```bash
curl -i -X PATCH $B/vehicles/1/approve -H "Authorization: Bearer $TS"
```

Esperado: `403 Forbidden` · `el rol no tiene permiso para esta operación`.
Está autenticada (no es 401), pero su rol no la autoriza.

**7.2 Login del administrador**

```bash
TA=$(curl -s -X POST $B/auth/login -H "$J" \
  -d '{"email":"admin@automarket.co","password":"AdminSegura123"}' | sed 's/.*"token":"\([^"]*\)".*/\1/')
echo $TA
```

**7.3 El administrador intenta publicar un vehículo**

```bash
curl -i -X POST $B/vehicles -H "Authorization: Bearer $TA" -H "$J" \
  -d '{"brand":"Kia","model":"Rio","year":2019,"price":40000000,"mileage":50000}'
```

Esperado: `403 Forbidden`. Publicar es exclusivo del rol vendedor.

## Paso 8 · HU-ADM-01 · Aprobar la publicación

```bash
curl -i -X PATCH $B/vehicles/1/approve -H "Authorization: Bearer $TA"
```

Esperado: `200 OK` y **`"status":"publicada"`** (criterio de aceptación de HU-ADM-01).

Si se intenta aprobar otra vez, el resultado es `409 Conflict`: solo se aprueban
publicaciones pendientes.

## Paso 9 · HU-VIS-01 · El visitante consulta el catálogo y el contacto

Sin token, como cualquier visitante:

```bash
curl -i $B/vehicles
curl -i $B/vehicles/1
```

Esperado: `200 OK`, el vehículo y el bloque del vendedor:

```json
"seller":{"name":"Ana Perez","email":"ana@mail.com","phone":"3001234567"}
```

Esto cumple el criterio de aceptación de HU-VIS-01.

## Paso 10 · HU-VEN-01 · Reportar la venta (solo el dueño)

**10.1 Crear un segundo vendedor, Luis (id 3), y guardar su token**

```bash
curl -s -X POST $B/auth/register -H "$J" \
  -d '{"name":"Luis Gomez","email":"luis@mail.com","phone":"3109999999","password":"Vendedor456"}'
TL=$(curl -s -X POST $B/auth/login -H "$J" \
  -d '{"email":"luis@mail.com","password":"Vendedor456"}' | sed 's/.*"token":"\([^"]*\)".*/\1/')
```

**10.2 Luis intenta marcar como vendido el carro de Ana**

```bash
curl -i -X PATCH $B/vehicles/1/sold -H "Authorization: Bearer $TL"
```

Esperado: `403 Forbidden` · `el recurso pertenece a otro usuario`.
Tiene el rol correcto, pero no es el dueño del recurso.

**10.3 Ana reporta su venta**

```bash
curl -i -X PATCH $B/vehicles/1/sold -H "Authorization: Bearer $TS"
curl -i $B/vehicles
```

Esperado: `200 OK` con `"status":"vendida"`, y después el catálogo vacío (`[]`):
el vehículo vendido sale del catálogo público.

## Paso 11 · HU-ADM-01 · Eliminar publicaciones y usuarios

**11.1 Luis publica un vehículo (id 2) y el administrador lo elimina**

```bash
curl -s -X POST $B/vehicles -H "Authorization: Bearer $TL" -H "$J" \
  -d '{"brand":"Mazda","model":"3","year":2020,"price":60000000,"mileage":30000}'
curl -i -X DELETE $B/vehicles/2 -H "Authorization: Bearer $TA"
```

Esperado: `204 No Content`.

**11.2 El administrador elimina al vendedor Luis**

```bash
curl -i -X DELETE $B/users/3 -H "Authorization: Bearer $TA"
```

Esperado: `204 No Content`. También se eliminan sus publicaciones.

**11.3 El token de Luis deja de servir**

```bash
curl -i -X POST $B/vehicles -H "Authorization: Bearer $TL" -H "$J" -d '{}'
```

Esperado: `401 Unauthorized`. Un usuario eliminado ya no se puede autenticar.

**11.4 Un vendedor no puede eliminar usuarios**

```bash
curl -i -X DELETE $B/users/2 -H "Authorization: Bearer $TS"
```

Esperado: `403 Forbidden`.

## Paso 12 · RF-01 Accounting · todo queda registrado

En la terminal 2, desde la carpeta del proyecto:

```bash
cat data/audit.log
```

Cada línea es una operación. Ejemplo (acortado):

```json
{"seq":6,"timestamp":"2026-10-07T03:47:33Z","actor_id":0,"actor_role":"visitante",
 "client_ip":"127.0.0.1","method":"POST","operation":"/vehicles","resource":"/vehicles",
 "status":401,"prev_hash":"dd42ce5a…","hash":"2401b4c2…"}
```

Qué revisar:

- Están **todas** las peticiones de la guía, también las públicas y los 401, 403 y 409.
- `actor_id` y `actor_role` indican quién hizo cada una (`visitante` si no había sesión).
- `prev_hash` de cada línea es igual al `hash` de la línea anterior: es la cadena.

## Paso 13 · RF-01 Accounting · la bitácora es íntegra

1. En la terminal 1 detén el servidor con `Ctrl + C`.
2. Altera la bitácora; por ejemplo, cambia el código de la línea 6 (el intento de publicar sin token del paso 5.1) de `401` a `200`:

   ```bash
   sed -i '6s/"status":401/"status":200/' data/audit.log
   ```

   También puedes abrirla en un editor y borrar o modificar cualquier línea.
3. Vuelve a arrancar con `go run .` (las variables del paso 2 siguen definidas en
   esa terminal).

Esperado: el servidor **se niega a arrancar** y muestra

```
abrir bitácora: la bitácora de auditoría fue alterada: entrada 6
```

Para seguir trabajando, deshaz el cambio
(`sed -i '6s/"status":200/"status":401/' data/audit.log`) o borra la carpeta `data`.

## Paso 14 · Contraseñas protegidas

```bash
cat data/users.json
```

Esperado: ninguna contraseña aparece en texto plano. Cada una se ve así:
`"password_hash": "pbkdf2-sha256$600000$<sal>$<clave>"`.

## Resumen de resultados esperados

| Paso | Qué se demuestra | Código esperado |
|---|---|---|
| 4.1 | Registro autónomo de vendedor | 201 |
| 4.2 · 4.3 · 4.4 | Correo duplicado · datos inválidos · login fallido | 409 · 400 · 401 |
| 5 | Authentication: sin token o con token falso | 401 |
| 6 | Publicación en estado `pendiente` | 201 |
| 7 | Authorization por rol | 403 |
| 8 | Aprobación → `publicada` | 200 |
| 9 | Catálogo público con contacto del vendedor | 200 |
| 10 | Venta solo por el dueño → `vendida` | 403 / 200 |
| 11 | Eliminar publicación y usuario | 204 / 401 / 403 |
| 12 | Toda operación registrada | — |
| 13 | Bitácora alterada → el servidor no arranca | — |
| 14 | Contraseñas con PBKDF2 | — |

## Paso 15 · Detener y limpiar

- En la terminal 1, `Ctrl + C` detiene el servidor.
- `rm -rf data` borra usuarios, vehículos y bitácora para repetir la guía desde
  cero. La carpeta `data/` está en `.gitignore` y nunca se sube al repositorio.

## Paso 16 · Entrega en GitHub

Repositorio: <https://github.com/reidermunoz/ciberseguridadTallerAAA>

| Rama | Contenido |
|---|---|
| `main` | Estructura inicial (rama base del Pull Request) |
| `feature/aaa` | Implementación completa |

**16.1 Subir cambios nuevos (si los hay)**

```bash
git checkout feature/aaa
git status                      # revisa qué cambió
git add -A
git commit -m "Describe el cambio"
git push
```

**16.2 Abrir el Pull Request**

1. Entra a <https://github.com/reidermunoz/ciberseguridadTallerAAA/pull/new/feature/aaa>.
2. Verifica **base: `main`** ← **compare: `feature/aaa`**.
3. Escribe el título y la descripción (qué HU y qué parte del RF-01 cubre) y pulsa
   **Create pull request**.
4. Agrega en la descripción del PR el enlace al video de demostración.

**16.3 Invitar al docente**

1. En el repositorio, entra a **Settings → Collaborators → Add people**.
2. Busca el usuario **`imcabezas`** y envía la invitación.

**16.4 Obtener el hash del commit para el PDF**

```bash
git checkout feature/aaa
git log -1 --format=%H
```

Copia el hash completo (40 caracteres). Si haces otro commit, el hash cambia:
toma siempre el del **último** commit antes de entregar.

**16.5 Qué va en el PDF de entrega**

- [ ] Portada
- [ ] Integrantes (Apellido – Nombre, en orden alfabético)
- [ ] URL del repositorio y hash del commit (16.4)
- [ ] Constancia de la invitación a `imcabezas` (16.3)
- [ ] URL del video de demostración/sustentación
- [ ] 1 diagrama de estructura y 1 diagrama de comportamiento
- [ ] Declaración de uso responsable de IA
