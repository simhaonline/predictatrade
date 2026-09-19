import { redirect } from "next/navigation";

/**
 * /signup → /register permanent redirect.
 *
 * The canonical registration route is /register (all internal links point
 * there), but the USER_GUIDE, marketing pages and shared links have always
 * advertised /signup. This keeps those inbound links working instead of a
 * 404. (Found during the 2026-09-19 endpoint audit: /signup returned 404.)
 */
export default function SignupRedirect() {
  redirect("/register");
}