Develop a complete Go 1.15 web application for allowing students
to register their in-class participation in a course via instructor-supplied
printed QR codes handed to them.

The specifications are described below.

# Functional specifications

## All pages
- HTML header `<title>` PixelBonus
- Sans Serif font
- On the left display the word PixelBonus, linking to the home page.
- On the right display 
  - the currently selected interface language and a pulldown menu to change it
    (English or Greek).  Language is persisted using an implementadion-defined
    cookie.
  - The currently logged in user (if logged in) with pulldown menu options
    to log out and visit the user's profile page.
  - At the page bottom links to
    - the project on GitHub https://github.com/dspinellis/pixelbonus with GitHub's icon
    - Link to the privacy policy provided here as `privacy_policy.md` but
      converted into HTML. (This can contain placeholders. Just document
      them in the README file.)

## Home page under pixelbonus.com
- Appears in the browser's preferred language; English by default.
Heading: Your participation counts!

Page split in half:
- Large (about 25% of screen size) student icon, "I'm a participant"
"Here you can redeem your personal QR code with only 2 clicks. Enter the code and your personal identifier and your participation will be instantly counted."
- Large icon of a professor, "I'm an instructor"
"With PixelBonus, you can reward active participation of the students by providing personal QR codes that they can redeem quickly and easily. Create an activity/course, print the codes and form rankings using several grading models."

Links to the two functionalities are provided by buttons.

## Instructor features

### Non-logged in instructor
Login screen with the following elements.
- Email (field)
- Password (field)
- Login (button)
- I forgot my password (button)
- Register (button)

### Email-based instructor registration
  - Provide full name, (unique) email, password
  - Token confirmation via email containing a validation URL with a random
    nonce valid for 24 hours.
  - When URL is accessed email is considered verified and only then
    the user can log in via an offered link to the login page.

### Password recovery
- Email
- Button: "Email me a link to set up a new password."
- Acknowledgment message discloses account's existence.
- Email contains a URL based on a suitably encoded random nonce
  that leads to a page where a new password can be specified.
- Link is valid for one hour.

### Profile page to change password

### Login
For instructors only, with email and password.

### Course functionality
List of existing courses starting with option to add a new course

- Courses are listed with most recently created ones the top.
- On the left of each course the following links are provided through button
  icons with bubble help.
   - Produce a new set of QR coupons
   - View course's grades
   - Download course's grades in CSV format
   - Public page of redeemed tokens
   - Settings

### Course settings page
- Course name
- Grading method (radio buttons) based on linear interpolation between
  participation distribution quartiles.

  - 0 = 0, ≥Q2 = 10
  - 0 = 0, ≥Q3 = 10
  - 0 = 0, Q4 = 10
  - Q2 = 5, ≥Q3 = 10
  - Q2 = 5, Q4 = 10
  - Q1 = 5, ≥Q2 = 10
  - Q1 = 5, ≥Q3 = 10
  - Q1 = 5, Q4 = 10
- Save button

### Creation of a new course
  - Course name (with suggestion to include the year or semester)
  - Grading method
  - Create button

### Produce QR code tokens with the following flow:
  - Click on the corresponding course's icon.
  - User provides number of required QR code tokens in multiples of 20.
  - Program acknowledges action with a message indicating that the
    user will be notified when coupons are ready.
  - Program initiates asynchronous generation of coupons.
  - Program creates a set with the specified number of 160 bit token
    identifiers.
  - Adds generated token identifiers in the database, associated with the
    selected course, a timestamp of the token set creation time,
    and marked as unused.
  - Creates asynchronously A4 PDF with coupons associated
    with the tokens. Each page contains 20 coupons in 10 rows by 2 columns
    and on the margins cut marks for splitting the individual coupons.
  - Coupon size is determined by A4 page size with 1 cm margins.
  - Font size should be the largest that can allow displaying the coupon's
    elements.
    Each coupon contains:
    - a QR code (error correction level L) for redeeming the coupon with
      a URL of the form
      https://pixelbonus.com/r?c=CODE, where CODE is a URL-safe
      encoding of the token code.
    - On its right, the token's code printed in 10 space-separated
      groups of four-digit hex characters in a fixed-width font.
    - Below the printed code the string pixelbonus.com in a larger font.
  - The PDF is created in `/var/tmp/pixelbonus/BASENAME.pdf`, where
    BASENAME is derived by hex-encoding a 128-bit random nonce.
    The directory shall be created if missing.
  - When the PDF is created, it emails to the instructor a message to
    download the generated file through a secure link URL based on the
    file name. This can be used by anybody who has it, multiple times,
    and remains valid forever.
  - Deployment instructions provide a cron job to delete old files.

### View and download course participation grades with the following flow.
  - Click on the corresponding course's icon.
  - Calculated grades on a 0-10 scale based on the number of redeemed
    coupons and the specified grading method.
    Quartiles are determined based on the distribution of coupon
    redemptions, e.g. Q2 is the median coupong redemption value.
    Grade is capped at 10, and rounded to an integer.
    Assume sufficiently large class size and number of participations.
    Zero redemption students are unknown and therefore not taken into
    considaration.
    All other grading choices (e.g. ties, ranking type) are implementation
    defined.
  - Calculated rank based on the number of redeemed coupons.
  - Output contains ordered by student id:
    - Student id
    - Number of redemptions
    - Grade
    - Rank

### View public redemption page
  - This page is publicly accessible for each course
    without registration or login.
  - Use a URL-encoded nonce under /public as the URL
  - This page contains ordered by student id:
    - Student id
    - Number of redemptions

## Student features

Students never need to register or authenticate.
They can redeem an unlimited number of tokens per course, but each
token can only be redeemed once.

### Redeem coupon through QR link
- The landing page contains
  - The course name (read-only)
  - The coupon code (read-only)
  - A field for inputting the student id
  - A button for redeeming the coupon

Responses can be:
  - Coupon (coupon number) for course (name) and student identifier (id)
   successfully redeemed.
  - Coupon (coupon number) for course (name) has been already redeemed.
  - Invalid coupon identifier (coupon number).

### Redeem code through identifier
- The page contains
  - A field for inputting the coupon code in hex.
    Spaces (separators) are allowed and ignored, case is normalized,
    invalid characters are not allowed to be typed.
  - A field for inputting the student id
  - A button for redeeming the coupon

Responses can be:
  - Coupon (coupon number) for course (name) and student identifier (id)
   successfully redeemed.
  - Coupon (coupon number) for course (name) has been already redeemed.
  - Invalid coupon identifier (coupon number).


# Non-functional constraints:

- Single static binary
- Only standard library unless absolutely necessary (e.g. for QR code
  generation)
- SQLite database
- No ORM
- No external JS frameworks
- Designed for Debian stable
- systemd-friendly
- Stand-alone installation (no need for a reverse proxy)
- Interface in English and Greek with ability to select language
- Language strings stored in a way that helps localisation
- Main palette color: 18bc9c, secondary: 2c3e50
- Automatic initialization of database schema
- During design specify readable URLs for all routes
- Email can be send with the Unix mail command
- Mail sender should be no-reply@pixelbonus.com
- Student identifiers are arbitrary strings (up to 20 characters)
  with no linkage between courses.
- Log actions to the file `pixelbonus.log` in the current directory.
- Read optional configuration file specified in an environment variable
  for port to listen, log file directory, and template directory.

## Localization and internationalization
Internationalization is REQUIRED.

- Every user-visible string MUST exist in both English and Greek.
- Hardcoded strings in templates are FORBIDDEN.
- All UI text must be loaded via a translation system.
- If any string appears only in English, the solution is invalid.

When no localized templates are used, employ a translation system with:
- translation files (e.g. en.json, el.json)
- key-based lookup (no inline strings)
- runtime language selection

## Security:
- TLS using letsencrypt certificate installed separately from the application
  using the files located in
  `/etc/letsencrypt/live/pixelbonus.com/{fullchain.pem,privkey.pem}`
  (These are external deployment prerequisites to be used from the specified
  locations. They are not subject to repo file-existence validation.)
- CSRF protection
- Password hashing (bcrypt or argon2)
- Secure session handling
- Input validation
- Login/logout with secure cookies

Deliverables:
- Full directory structure
- go.mod
- SQL schema
- Main.go
- Templates in both languages
- systemd service file
