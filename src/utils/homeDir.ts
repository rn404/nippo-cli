// Replaced with native APIs
const homeDir = (): string | undefined => {
  return Deno.env.get('HOME') || Deno.env.get('USERPROFILE');
};

export { homeDir };
