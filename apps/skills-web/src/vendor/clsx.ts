export default function clsx(...values: Array<string | false | null | undefined>): string {
  return values.filter(Boolean).join(' ');
}
