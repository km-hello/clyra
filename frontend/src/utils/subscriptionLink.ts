export function isSubscriptionLink(value: string): boolean {
  if (
    value.length > 2048 ||
    !/^https?:\/\/[^/?#]+/i.test(value) ||
    /^https?:\/\/[^/?#]*@/i.test(value) ||
    /[\s\\\p{Cc}]/u.test(value)
  )
    return false
  try {
    const url = new URL(value)
    return !!url.hostname && !url.username && !url.password
  } catch {
    return false
  }
}
