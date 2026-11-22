import type en from "../en/twofactor";

const dict = {
  tfaTitle: "Autenticação em dois fatores",
  tfaStatusTitle: "Status",
  tfaOn: "A autenticação em dois fatores está ligada: o login pede um código do seu app autenticador.",
  tfaOff: "Adicione uma segunda etapa ao seu login: um código de um app autenticador no celular.",
  tfaEnable: "Ativar a autenticação em dois fatores",
  tfaSetupTitle: "Configurar",
  tfaScan:
    "Leia este QR code com seu app autenticador, ou digite a chave de configuração à mão, e depois digite o código que ele mostra.",
  tfaQrAlt: "QR code de configuração dos dois fatores",
  tfaSetupKey: "Chave de configuração",
  tfaCode: "Código",
  tfaConfirm: "Confirmar",
  tfaRecoveryTitle: "Códigos de recuperação",
  tfaRecoveryDescription: "Códigos de uso único para o dia em que você perder o celular.",
  tfaHelpRecovery: "Sobre os códigos de recuperação",
  tfaRecoveryStore:
    "Guarde estes códigos de recuperação em um lugar seguro. Cada um vale uma vez se você perder o aparelho; eles não são mostrados de novo.",
  tfaCopyCodes: "Copiar códigos",
  tfaCopied: "Copiado",
  tfaRecoveryLeft: "Restam {count} códigos de recuperação.",
  tfaCurrentPassword: "Senha atual",
  tfaRegenerate: "Gerar novos códigos de recuperação",
  tfaTurnOff: "Desligar",
  tfaTurnOffDescription: "O login volta a usar só a senha. Você pode ligar de novo quando quiser.",
  tfaDisable: "Desativar a autenticação em dois fatores",
  tfaBack: "Voltar ao painel",
} satisfies Record<keyof typeof en, string>;
export default dict;
