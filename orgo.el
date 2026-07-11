;;; orgo.el --- Browse org-roam notes in the browser -*- lexical-binding: t; -*-

;; Author: naok-000
;; URL: https://github.com/naok-000/orgo
;; Version: 0.1.0
;; Package-Requires: ((emacs "27.1"))
;; Keywords: outlines, hypermedia, org-roam

;; This file is not part of GNU Emacs.

;;; Commentary:

;; orgo serves your org-roam directory as a web application: notes are
;; rendered to HTML, `id:' links are followable, and the link structure
;; is visualized as an interactive graph.  Notes open in switchable tabs.
;;
;; The heavy lifting is done by the standalone `orgo' binary (Go); this
;; package only manages that process from Emacs.  Install the binary
;; with one of:
;;
;;   go install github.com/naok-000/orgo@latest
;;   nix profile install github:naok-000/orgo
;;
;; Then install this package straight from GitHub:
;;
;;   ;; Emacs 30+
;;   (use-package orgo
;;     :vc (:url "https://github.com/naok-000/orgo" :rev :newest))
;;
;;   ;; Emacs 29 with vc-use-package, or straight.el, work the same way.
;;
;; Entry points:
;;
;;   M-x orgo        start the server (if needed) and open the browser
;;   M-x orgo-open-current-note   show the org-roam node at point
;;   M-x orgo-stop   stop the server

;;; Code:

(require 'browse-url)
(require 'url-util)

(declare-function org-entry-get "org" (epom property &optional inherit literal-nil))

(defgroup orgo nil
  "Browse org-roam notes in the browser."
  :group 'org
  :prefix "orgo-")

(defcustom orgo-executable "orgo"
  "Name or path of the orgo binary."
  :type 'string)

(defcustom orgo-directory nil
  "Directory of org-roam notes to serve.
When nil, fall back to `org-roam-directory', then to \"~/org-roam\"."
  :type '(choice (const :tag "Auto-detect" nil) directory))

(defcustom orgo-port 35911
  "Port the orgo server listens on."
  :type 'natnum)

(defcustom orgo-extra-args nil
  "Extra command line arguments passed to the orgo binary."
  :type '(repeat string))

(defvar orgo--process nil
  "The running orgo server process, if any.")

(defun orgo--directory ()
  "Return the directory orgo should serve."
  (expand-file-name
   (or orgo-directory
       (and (boundp 'org-roam-directory) org-roam-directory)
       "~/org-roam")))

(defun orgo--url (&optional path)
  "Return the orgo UI url, with optional PATH appended."
  (format "http://127.0.0.1:%d/%s" orgo-port (or path "")))

(defun orgo--running-p ()
  "Non-nil when the orgo server process is alive."
  (and orgo--process (process-live-p orgo--process)))

(defun orgo--wait-ready (&optional timeout)
  "Wait until the orgo server accepts connections, up to TIMEOUT seconds.
Returns non-nil on success."
  (let ((deadline (+ (float-time) (or timeout 15)))
        ready)
    (while (and (not ready) (< (float-time) deadline) (orgo--running-p))
      (condition-case nil
          (let ((proc (open-network-stream "orgo-ping" nil "127.0.0.1" orgo-port)))
            (delete-process proc)
            (setq ready t))
        (error (sleep-for 0.2))))
    ready))

;;;###autoload
(defun orgo-start ()
  "Start the orgo server for `orgo-directory'.
Signals an error when the binary cannot be found."
  (interactive)
  (if (orgo--running-p)
      (message "orgo already running at %s" (orgo--url))
    (let ((bin (executable-find orgo-executable))
          (dir (orgo--directory)))
      (unless bin
        (error "Cannot find `%s'; install it with: go install github.com/naok-000/orgo@latest"
               orgo-executable))
      (unless (file-directory-p dir)
        (error "orgo: directory %s does not exist" dir))
      (setq orgo--process
            (apply #'start-process "orgo" " *orgo*"
                   bin
                   "--port" (number-to-string orgo-port)
                   "--no-browser"
                   (append orgo-extra-args (list dir))))
      (set-process-query-on-exit-flag orgo--process nil)
      (message "orgo serving %s at %s" dir (orgo--url)))))

;;;###autoload
(defun orgo-stop ()
  "Stop the orgo server."
  (interactive)
  (if (orgo--running-p)
      (progn
        (interrupt-process orgo--process)
        (setq orgo--process nil)
        (message "orgo stopped"))
    (message "orgo is not running")))

;;;###autoload
(defun orgo ()
  "Start the orgo server if needed and open the UI in the browser."
  (interactive)
  (unless (orgo--running-p)
    (orgo-start))
  (unless (orgo--wait-ready)
    (user-error "orgo server did not become ready; see buffer *orgo*"))
  (browse-url (orgo--url)))

;;;###autoload
(defun orgo-open-current-note ()
  "Open the org-roam node at point in the orgo UI."
  (interactive)
  (unless (derived-mode-p 'org-mode)
    (user-error "Not in an org buffer"))
  (require 'org)
  (let ((id (org-entry-get nil "ID" t)))
    (unless id
      (user-error "No :ID: property found at point or above"))
    (unless (orgo--running-p)
      (orgo-start))
    (unless (orgo--wait-ready)
      (user-error "orgo server did not become ready; see buffer *orgo*"))
    (browse-url (orgo--url (concat "#/note/" (url-hexify-string id))))))

(provide 'orgo)
;;; orgo.el ends here
