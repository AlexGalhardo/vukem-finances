import type en from "../en/sync";

const dict = {
  devices: "Dispositivos",
  devicesIntro:
    "Os celulares que sincronizam este workspace com o servidor. Um celular se registra com um token de API que tenha o escopo sync.",
  devicesListTitle: "Seus dispositivos neste workspace",
  devicesNone: "Nenhum dispositivo sincroniza este workspace ainda.",
  devicesName: "Nome",
  devicesPlatform: "Plataforma",
  devicesLastSeen: "Visto por último",
  devicesLastPull: "Última leitura (número da mudança)",
  devicesNeverSeen: "nunca",
  devicesStatus: "Situação",
  devicesStatusLive: "Sincronizando",
  devicesStatusRevoked: "Revogado",
  devicesRevoke: "Revogar",
  devicesRevokeNote:
    "Revogar um dispositivo interrompe a sincronização na hora. O dispositivo mantém os dados locais até você limpá-los nele. Revogar um token de API também revoga os dispositivos dele.",
} satisfies Record<keyof typeof en, string>;
export default dict;
