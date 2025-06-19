# 🔒 Guía de Seguridad para Variables de Entorno

## ⚠️ **REGLAS DE ORO - NUNCA HACER ESTO:**

❌ **NUNCA** subir credenciales a GitHub  
❌ **NUNCA** poner passwords en archivos de código  
❌ **NUNCA** compartir DATABASE_URL en texto plano  
❌ **NUNCA** commitear archivos .env con datos reales  

## ✅ **MEJORES PRÁCTICAS:**

### **1. Desarrollo Local (tu laptop)**
```bash
# Crear archivo .env local (NO subirlo a Git)
echo "DATABASE_URL=postgresql://edwin:..." > app.env
echo "JWT_SECRET=tu-secret-local" >> app.env

# Asegurar que esté en .gitignore
echo "app.env" >> .gitignore
```

### **2. Producción en Render**
- ✅ Variables sensibles **SOLO** en Dashboard de Render
- ✅ Variables no-sensibles pueden ir en render.yaml
- ✅ Usar "Generate Value" para secrets automáticos

### **3. Alternativas Seguras para tu Base de Datos**

#### **Opción A: Dashboard Manual (Recomendado)**
1. Deploy sin DATABASE_URL
2. Configurar manualmente en Dashboard
3. Redeploy automático

#### **Opción B: Conectar Base de Datos Existente**
```yaml
# En render.yaml - referencia segura
envVars:
  - key: DATABASE_URL
    fromDatabase:
      name: tu-db-existente
      property: connectionString
```

#### **Opción C: Environment Groups**
1. Crear "Environment Group" en Render
2. Agregar variables sensibles ahí
3. Referenciar el grupo en el servicio

## 🛡️ **Verificación de Seguridad**

### **Antes de hacer git commit:**
```bash
# Verificar que no hay credenciales
grep -r "postgresql://" . --exclude-dir=.git
grep -r "password" . --exclude-dir=.git --exclude="*.md"

# Si encuentra algo sensible, revisar y remover
```

### **Archivos que SÍ pueden ir a Git:**
- ✅ `render.yaml` (sin credenciales)
- ✅ `app.env.example` (valores de ejemplo)
- ✅ `app.env.production` (solo comentarios/referencias)
- ✅ Scripts de deployment

### **Archivos que NUNCA van a Git:**
- ❌ `app.env` (con datos reales)
- ❌ `.env` (con datos reales)
- ❌ Cualquier archivo con credenciales reales

## 🔧 **Configuración Paso a Paso (Segura)**

### **1. Preparar repositorio (sin credenciales)**
```bash
# Verificar .gitignore
cat .gitignore | grep -E "(\.env|app\.env)$"

# Si no están, agregarlos
echo "app.env" >> .gitignore
echo ".env" >> .gitignore

# Commit y push (seguro)
git add .
git commit -m "feat: Backend listo para deployment seguro"
git push
```

### **2. Deployment en Render**
```bash
# 1. Conectar repositorio GitHub
# 2. Render detecta render.yaml automáticamente
# 3. Variables no-sensibles se aplican automáticamente
# 4. Fallarán las sensibles (esperado)
```

### **3. Configurar variables sensibles**
```bash
# En Render Dashboard > Environment:
DATABASE_URL: [Pegar tu URL real aquí]
JWT_SECRET: [Hacer clic en "Generate Value"]
```

### **4. Verificar funcionamiento**
```bash
# Health check
curl https://tu-app.onrender.com/api/v1/health

# Debería responder: {"status": "ok", ...}
```

## 🎯 **¿Por qué es importante?**

1. **Ciberseguridad**: Credenciales en GitHub son públicas
2. **Compliance**: Muchas empresas requieren estas prácticas
3. **Profesionalismo**: Demuestra conocimiento de seguridad
4. **Escalabilidad**: Facilita manejo de múltiples entornos

## 🚨 **¿Qué hacer si ya subiste credenciales?**

1. **Cambiar credenciales inmediatamente**
2. **Borrar historial de Git** (git filter-branch)
3. **Crear nuevas credenciales**
4. **Implementar esta guía correctamente**

---

**Recuerda: La seguridad no es opcional - es fundamental** 🔐
