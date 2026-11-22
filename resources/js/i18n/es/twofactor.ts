import type en from "../en/twofactor";

const dict = {
  tfaTitle: "Autenticación en dos pasos",
  tfaStatusTitle: "Estado",
  tfaOn: "La autenticación en dos pasos está activada: al iniciar sesión se pide un código de tu app de autenticación.",
  tfaOff: "Añade un segundo paso a tu inicio de sesión: un código de una app de autenticación en tu teléfono.",
  tfaEnable: "Activar la autenticación en dos pasos",
  tfaSetupTitle: "Configurar",
  tfaScan:
    "Escanea este código QR con tu app de autenticación, o escribe la clave de configuración a mano, y luego escribe el código que muestra.",
  tfaQrAlt: "Código QR de configuración de los dos pasos",
  tfaSetupKey: "Clave de configuración",
  tfaCode: "Código",
  tfaConfirm: "Confirmar",
  tfaRecoveryTitle: "Códigos de recuperación",
  tfaRecoveryDescription: "Códigos de un solo uso para el día en que pierdas el teléfono.",
  tfaHelpRecovery: "Acerca de los códigos de recuperación",
  tfaRecoveryStore:
    "Guarda estos códigos de recuperación en un lugar seguro. Cada uno sirve una vez si pierdes el dispositivo; no se vuelven a mostrar.",
  tfaCopyCodes: "Copiar códigos",
  tfaCopied: "Copiado",
  tfaRecoveryLeft: "Te quedan {count} códigos de recuperación.",
  tfaCurrentPassword: "Contraseña actual",
  tfaRegenerate: "Generar nuevos códigos de recuperación",
  tfaTurnOff: "Desactivar",
  tfaTurnOffDescription:
    "El inicio de sesión vuelve a usar solo la contraseña. Puedes activarlo de nuevo cuando quieras.",
  tfaDisable: "Desactivar la autenticación en dos pasos",
  tfaBack: "Volver al panel",
} satisfies Record<keyof typeof en, string>;
export default dict;
