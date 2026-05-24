export const isPromoterVerifyCodeValid = (value) => {
  return /^\d{6}$/.test(String(value || '').trim())
}
