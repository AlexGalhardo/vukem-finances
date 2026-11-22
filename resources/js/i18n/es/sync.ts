import type en from "../en/sync";

const dict = {
  devices: "Dispositivos",
  devicesIntro:
    "Los teléfonos que sincronizan este workspace con el servidor. Un teléfono se registra con un token de API que tenga el alcance sync.",
  devicesListTitle: "Tus dispositivos en este workspace",
  devicesNone: "Ningún dispositivo sincroniza este workspace todavía.",
  devicesName: "Nombre",
  devicesPlatform: "Plataforma",
  devicesLastSeen: "Visto por última vez",
  devicesLastPull: "Última lectura (número de cambio)",
  devicesNeverSeen: "nunca",
  devicesStatus: "Estado",
  devicesStatusLive: "Sincronizando",
  devicesStatusRevoked: "Revocado",
  devicesRevoke: "Revocar",
  devicesRevokeNote:
    "Revocar un dispositivo detiene su sincronización al instante. El dispositivo conserva sus datos locales hasta que los borres en él. Revocar un token de API también revoca sus dispositivos.",
  devicesAdd: "Añadir dispositivo",
  devicesPairTitle: "Emparejar un teléfono",
  devicesPairBody:
    "En la app, abre Conectar y escribe la dirección del servidor y este código. El código sirve una vez y caduca en 10 minutos.",
  devicesPairServer: "Dirección del servidor",
  devicesPairCode: "Código de emparejamiento",
  devicesPairExpires: "Caduca a las",
  devicesPairCopy: "Copiar código",
  devicesPairDismiss: "Listo",
} satisfies Record<keyof typeof en, string>;
export default dict;
