# 🚀 Deploy en Render

Esta guía te ayudará a hacer deploy de tu API de Hellfire Gatekeeper en Render.

## 📋 Prerrequisitos

1. **Cuenta en Render**: [render.com](https://render.com)
2. **Cuenta en Cloudinary**: [cloudinary.com](https://cloudinary.com) (para storage de imágenes)
3. **Repositorio en GitHub**: Tu código debe estar en GitHub

## 🔧 Configuración

### 1. Variables de Entorno en Render

En el dashboard de Render, configura estas variables de entorno:

#### Base de Datos (se configuran automáticamente)
- `MYSQL_USER` - Se configura automáticamente desde la base de datos
- `MYSQL_PASSWORD` - Se configura automáticamente desde la base de datos
- `MYSQL_DATABASE` - Se configura automáticamente desde la base de datos
- `DB_HOST` - Se configura automáticamente desde la base de datos
- `DB_PORT` - Se configura automáticamente desde la base de datos

#### JWT (se configura automáticamente)
- `JWT_SECRET` - Se genera automáticamente
- `JWT_EXPIRATION_MINUTES` - Se configura automáticamente a "60"

#### Cloudinary (configurar manualmente)
- `CLOUDINARY_CLOUD_NAME` - Tu cloud name de Cloudinary
- `CLOUDINARY_API_KEY` - Tu API key de Cloudinary
- `CLOUDINARY_API_SECRET` - Tu API secret de Cloudinary

#### Servidor
- `PORT` - Se configura automáticamente a "10000"

### 2. Pasos para el Deploy

1. **Conecta tu repositorio de GitHub a Render**
2. **Crea una base de datos MySQL en Render**
3. **Crea un servicio web en Render** usando el archivo `render.yaml`
4. **Configura las variables de entorno de Cloudinary**
5. **Haz el deploy**

## 🗂️ Estructura de Archivos para Deploy

```
├── render.yaml          # Configuración de Render
├── start.sh            # Script de inicio (migraciones + servidor)
├── cmd/
│   ├── api/            # Aplicación principal
│   └── migrate/        # Script de migraciones
├── migrations/         # Archivos de migración SQL
└── internal/           # Código de la aplicación
```

## 🔄 Flujo de Deploy

1. **Build**: Render compila la aplicación y el script de migraciones
2. **Start**: Se ejecuta `start.sh` que:
   - Ejecuta las migraciones de base de datos
   - Inicia el servidor API

## 📸 Storage de Imágenes

El proyecto está configurado para usar **Cloudinary** como storage de imágenes:

- **Ventajas**: 25GB gratis, CDN global, optimización automática
- **Fallback**: Si no hay credenciales de Cloudinary, usa storage local (se pierde en cada deploy)

## 🧪 Testing Local

Para probar localmente con Cloudinary:

1. Crea un archivo `.env` con tus credenciales
2. Ejecuta: `go run ./cmd/migrate` (migraciones)
3. Ejecuta: `go run ./cmd/api` (servidor)

## 🚨 Troubleshooting

### Error de Conexión a Base de Datos
- Verifica que la base de datos MySQL esté creada en Render
- Revisa que las variables de entorno estén configuradas

### Error de Migraciones
- Verifica que el archivo `migrations/` esté en el repositorio
- Revisa los logs de Render para ver errores específicos

### Error de Cloudinary
- Verifica que las credenciales de Cloudinary estén configuradas
- Revisa que tu cuenta de Cloudinary esté activa

## 📞 Soporte

Si tienes problemas con el deploy, revisa:
1. Los logs de Render en el dashboard
2. Las variables de entorno
3. La configuración de la base de datos
4. Las credenciales de Cloudinary
