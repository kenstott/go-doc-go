"""
Exchange Content Source for the document pointer system.

This module provides integration with Microsoft Exchange servers via EWS and Graph API.
"""

import logging
import re
from typing import Dict, Any, List, Optional, TYPE_CHECKING, Set
import time
import base64
from urllib.parse import quote, urljoin

from .base import ContentSource

# Import types for type checking only - these won't be imported at runtime
if TYPE_CHECKING:
    import requests
    from requests import Session, Response
    from bs4 import BeautifulSoup
    import dateutil.parser
    from datetime import datetime, timezone
    from exchangelib import DELEGATE, Account, Credentials, Configuration, EWSDateTime
    from exchangelib.protocol import BaseProtocol, NoVerifyHTTPAdapter
    import msal

    # Define type aliases for type checking
    RequestsSessionType = Session
    RequestsResponseType = Response
    BeautifulSoupType = BeautifulSoup
    DateUtilParserType = dateutil.parser
    DatetimeType = datetime
    ExchangeAccountType = Account
    MSALClientType = msal.ConfidentialClientApplication
else:
    # Runtime type aliases - use generic Python types
    RequestsSessionType = Any
    RequestsResponseType = Any
    BeautifulSoupType = Any
    DateUtilParserType = Any
    DatetimeType = Any
    ExchangeAccountType = Any
    MSALClientType = Any

logger = logging.getLogger(__name__)

# Define global flags for availability - these will be set at runtime
REQUESTS_AVAILABLE = False
BS4_AVAILABLE = False
DATEUTIL_AVAILABLE = False
EXCHANGELIB_AVAILABLE = False
MSAL_AVAILABLE = False

# Try to import requests conditionally
try:
    import requests

    REQUESTS_AVAILABLE = True
except ImportError:
    logger.warning("requests not available. Install with 'pip install requests' to use Exchange content source.")

# Try to import BeautifulSoup conditionally
try:
    from bs4 import BeautifulSoup

    BS4_AVAILABLE = True
except ImportError:
    logger.warning("beautifulsoup4 not available. Install with 'pip install beautifulsoup4' for improved HTML parsing.")

# Try to import dateutil conditionally
try:
    import dateutil.parser
    from datetime import datetime, timezone

    DATEUTIL_AVAILABLE = True
except ImportError:
    logger.warning(
        "python-dateutil not available. Install with 'pip install python-dateutil' for improved date handling.")

# Try to import exchangelib conditionally
try:
    from exchangelib import DELEGATE, Account, Credentials, Configuration, EWSDateTime
    from exchangelib.protocol import BaseProtocol, NoVerifyHTTPAdapter

    EXCHANGELIB_AVAILABLE = True
except ImportError:
    logger.warning("exchangelib not available. Install with 'pip install exchangelib' to use EWS integration.")

# Try to import msal conditionally
try:
    import msal

    MSAL_AVAILABLE = True
except ImportError:
    logger.warning("msal not available. Install with 'pip install msal' to use Graph API integration.")


class ExchangeContentSource(ContentSource):
    """Content source for Microsoft Exchange servers."""

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize the Exchange content source.

        Args:
            config: Configuration dictionary containing Exchange connection details
        """
        if not REQUESTS_AVAILABLE:
            raise ImportError("requests is required for ExchangeContentSource but not available")

        super().__init__(config)

        # Connection configuration
        self.server_url = config.get("server_url", "").rstrip('/')
        self.email_address = config.get("email_address", "")
        self.username = config.get("username", "")
        self.password = config.get("password", "")

        # API selection: 'ews', 'graph', or 'auto'
        self.api_type = config.get("api_type", "auto")

        # OAuth2/Graph API configuration
        self.client_id = config.get("client_id", "")
        self.client_secret = config.get("client_secret", "")
        self.tenant_id = config.get("tenant_id", "")
        self.authority = config.get("authority", f"https://login.microsoftonline.com/{self.tenant_id}")

        # EWS specific configuration
        self.ews_autodiscover = config.get("ews_autodiscover", True)
        self.ews_verify_ssl = config.get("ews_verify_ssl", True)

        # Content filtering configuration
        self.folders = config.get("folders", ["inbox"])  # List of folder names to scan
        self.mailbox = config.get("mailbox", None)  # Specific mailbox (for shared mailboxes)
        self.max_results = config.get("max_results", 100)
        self.days_back = config.get("days_back", 30)  # How many days back to fetch emails
        self.search_query = config.get("search_query", "")  # Exchange search query

        # Content inclusion configuration
        self.include_body = config.get("include_body", True)
        self.include_headers = config.get("include_headers", True)
        self.include_attachments = config.get("include_attachments", False)
        self.include_thread = config.get("include_thread", True)
        self.attachment_types = config.get("attachment_types", [])  # Filter by attachment types

        # Content format preferences
        self.prefer_html = config.get("prefer_html", True)
        self.include_original_headers = config.get("include_original_headers", False)

        # Link following configuration
        self.max_link_depth = config.get("max_link_depth", 1)
        self.follow_replies = config.get("follow_replies", True)
        self.follow_forwards = config.get("follow_forwards", True)

        # Initialize connections
        self.session: Optional[RequestsSessionType] = None
        self.ews_account: Optional[ExchangeAccountType] = None
        self.graph_client: Optional[MSALClientType] = None
        self.access_token: Optional[str] = None

        # Cache for content and tokens
        self.content_cache = {}
        self.token_cache = {}

        # Determine and initialize the appropriate API
        self._initialize_connection()

    def _initialize_connection(self):
        """Initialize connection to Exchange server based on configuration."""
        if self.api_type == "graph" or (self.api_type == "auto" and self.client_id):
            self._initialize_graph_api()
        elif self.api_type == "ews" or (self.api_type == "auto" and not self.client_id):
            self._initialize_ews()
        else:
            raise ValueError("Unable to determine Exchange connection method. Check your configuration.")

    def _initialize_graph_api(self):
        """Initialize Microsoft Graph API connection."""
        if not MSAL_AVAILABLE:
            raise ImportError("msal is required for Graph API but not available")

        if not self.client_id or not self.client_secret:
            raise ValueError("client_id and client_secret are required for Graph API")

        try:
            # Initialize MSAL client
            self.graph_client = msal.ConfidentialClientApplication(
                client_id=self.client_id,
                client_credential=self.client_secret,
                authority=self.authority
            )

            # Get access token
            self._refresh_graph_token()

            # Initialize requests session for Graph API calls
            self.session = requests.Session()

            logger.debug(f"Successfully initialized Graph API connection for: {self.email_address}")

        except Exception as e:
            logger.error(f"Error initializing Graph API: {str(e)}")
            raise

    def _initialize_ews(self):
        """Initialize Exchange Web Services connection."""
        if not EXCHANGELIB_AVAILABLE:
            raise ImportError("exchangelib is required for EWS but not available")

        if not self.username or not self.password:
            raise ValueError("username and password are required for EWS")

        try:
            # Set up credentials
            credentials = Credentials(username=self.username, password=self.password)

            # Configure connection
            if self.ews_autodiscover:
                # Use autodiscovery
                self.ews_account = Account(
                    primary_smtp_address=self.email_address,
                    credentials=credentials,
                    autodiscover=True,
                    access_type=DELEGATE
                )
            else:
                # Manual configuration
                if not self.server_url:
                    raise ValueError("server_url is required when autodiscovery is disabled")

                config = Configuration(
                    server=self.server_url,
                    credentials=credentials
                )

                if not self.ews_verify_ssl:
                    # Disable SSL verification if configured
                    BaseProtocol.HTTP_ADAPTER_CLS = NoVerifyHTTPAdapter

                self.ews_account = Account(
                    primary_smtp_address=self.email_address,
                    config=config,
                    autodiscover=False,
                    access_type=DELEGATE
                )

            logger.debug(f"Successfully initialized EWS connection for: {self.email_address}")

        except Exception as e:
            logger.error(f"Error initializing EWS: {str(e)}")
            raise

    def _refresh_graph_token(self):
        """Refresh Graph API access token."""
        if not self.graph_client:
            raise ValueError("Graph client not initialized")

        try:
            # Get token using client credentials flow
            result = self.graph_client.acquire_token_for_client(
                scopes=["https://graph.microsoft.com/.default"]
            )

            if "access_token" in result:
                self.access_token = result["access_token"]
                self.token_cache["expires_at"] = time.time() + result.get("expires_in", 3600)
                logger.debug("Successfully refreshed Graph API token")
            else:
                error_desc = result.get("error_description", "Unknown error")
                raise ValueError(f"Failed to acquire token: {error_desc}")

        except Exception as e:
            logger.error(f"Error refreshing Graph API token: {str(e)}")
            raise

    def _ensure_valid_token(self):
        """Ensure we have a valid Graph API token."""
        if not self.access_token:
            self._refresh_graph_token()
            return

        # Check if token is expired (with 5 minute buffer)
        expires_at = self.token_cache.get("expires_at", 0)
        if time.time() + 300 > expires_at:
            self._refresh_graph_token()

    def get_safe_connection_string(self) -> str:
        """Return a safe version of the connection string with credentials masked."""
        if self.api_type == "graph" or self.client_id:
            return f"Graph API: {self.email_address}"
        else:
            return f"EWS: {self.email_address}@{self.server_url}"

    def fetch_document(self, source_id: str) -> Dict[str, Any]:
        """
        Fetch email content from Exchange.

        Args:
            source_id: Identifier for the email (message ID or Graph ID)

        Returns:
            Dictionary containing email content and metadata

        Raises:
            ValueError: If Exchange is not configured or email not found
        """
        logger.debug(f"Fetching Exchange email: {source_id}")

        try:
            # Extract message ID from source_id
            message_id = self._extract_message_id(source_id)

            if self.api_type == "graph" or self.client_id:
                return self._fetch_document_graph(message_id)
            else:
                return self._fetch_document_ews(message_id)

        except Exception as e:
            logger.error(f"Error fetching Exchange email {source_id}: {str(e)}")
            raise

    def _fetch_document_graph(self, message_id: str) -> Dict[str, Any]:
        """Fetch email using Microsoft Graph API."""
        if not self.session:
            raise ValueError("Graph API not configured")

        self._ensure_valid_token()

        # Construct Graph API URL
        mailbox_part = f"users/{self.mailbox or self.email_address}"
        api_url = f"https://graph.microsoft.com/v1.0/{mailbox_part}/messages/{message_id}"

        # Set up headers
        headers = {
            "Authorization": f"Bearer {self.access_token}",
            "Content-Type": "application/json"
        }

        # Set up parameters for the API request
        params = {
            "$select": "id,subject,body,bodyPreview,sender,from,toRecipients,ccRecipients,bccRecipients,receivedDateTime,sentDateTime,hasAttachments,internetMessageId,conversationId,parentFolderId,importance,flag"
        }

        # Make API request
        response = self.session.get(api_url, headers=headers, params=params)
        response.raise_for_status()
        email_data = response.json()

        return self._process_email_data_graph(email_data)

    def _fetch_document_ews(self, message_id: str) -> Dict[str, Any]:
        """Fetch email using Exchange Web Services."""
        if not self.ews_account:
            raise ValueError("EWS not configured")

        # Find the message by message ID
        # EWS uses different ID formats, we'll search by internet message ID or EWS ID
        try:
            # Try to get message directly if it's an EWS item ID
            from exchangelib import Message

            if message_id.startswith("<") and message_id.endswith(">"):
                # This looks like an internet message ID
                messages = self.ews_account.inbox.filter(
                    message_id=message_id
                ).only('id', 'subject', 'body', 'sender', 'datetime_received', 'datetime_sent', 'has_attachments',
                       'message_id', 'conversation_id')

                if messages:
                    message = list(messages)[0]
                else:
                    raise ValueError(f"Message not found: {message_id}")
            else:
                # Try as EWS item ID
                message = Message.get(item_id=message_id, account=self.ews_account)

            return self._process_email_data_ews(message)

        except Exception as e:
            logger.error(f"Error fetching EWS message {message_id}: {str(e)}")
            raise

    def _process_email_data_graph(self, email_data: Dict[str, Any]) -> Dict[str, Any]:
        """Process email data from Graph API into standard format."""
        message_id = email_data.get("id", "")
        subject = email_data.get("subject", "")

        # Get email body
        body_data = email_data.get("body", {})
        body_content = body_data.get("content", "")
        body_type = body_data.get("contentType", "text").lower()

        # Get sender information
        sender_data = email_data.get("sender", {}) or email_data.get("from", {})
        sender_email = sender_data.get("emailAddress", {}).get("address", "")
        sender_name = sender_data.get("emailAddress", {}).get("name", "")

        # Get recipient information
        to_recipients = [r.get("emailAddress", {}).get("address", "") for r in email_data.get("toRecipients", [])]
        cc_recipients = [r.get("emailAddress", {}).get("address", "") for r in email_data.get("ccRecipients", [])]

        # Get timestamps
        received_time = email_data.get("receivedDateTime", "")
        sent_time = email_data.get("sentDateTime", "")

        # Build HTML content
        html_content = self._build_email_html(
            subject=subject,
            sender_name=sender_name,
            sender_email=sender_email,
            to_recipients=to_recipients,
            cc_recipients=cc_recipients,
            sent_time=sent_time,
            received_time=received_time,
            body_content=body_content,
            body_type=body_type
        )

        # Create fully qualified source identifier
        qualified_source = f"exchange://{self.email_address}/{message_id}"

        # Construct metadata
        metadata = {
            "message_id": message_id,
            "internet_message_id": email_data.get("internetMessageId", ""),
            "conversation_id": email_data.get("conversationId", ""),
            "subject": subject,
            "sender_email": sender_email,
            "sender_name": sender_name,
            "to_recipients": to_recipients,
            "cc_recipients": cc_recipients,
            "received_datetime": received_time,
            "sent_datetime": sent_time,
            "has_attachments": email_data.get("hasAttachments", False),
            "importance": email_data.get("importance", "normal"),
            "content_type": "html",
            "api_type": "graph"
        }

        # Generate content hash for change detection
        content_hash = self.get_content_hash(html_content)

        # Cache the content
        self.content_cache[message_id] = {
            "content": html_content,
            "metadata": metadata,
            "hash": content_hash,
            "last_accessed": time.time()
        }

        return {
            "id": qualified_source,
            "content": html_content,
            "doc_type": "html",
            "metadata": metadata,
            "content_hash": content_hash
        }

    def _process_email_data_ews(self, message) -> Dict[str, Any]:
        """Process email data from EWS into standard format."""
        message_id = message.id
        subject = message.subject or ""

        # Get email body
        body_content = ""
        body_type = "text"

        if hasattr(message, 'body') and message.body:
            body_content = message.body
            body_type = "html" if message.body_type == "HTML" else "text"

        # Get sender information
        sender_email = ""
        sender_name = ""
        if hasattr(message, 'sender') and message.sender:
            sender_email = message.sender.email_address
            sender_name = message.sender.name or sender_email

        # Get recipient information
        to_recipients = []
        cc_recipients = []

        if hasattr(message, 'to_recipients'):
            to_recipients = [r.email_address for r in message.to_recipients]
        if hasattr(message, 'cc_recipients'):
            cc_recipients = [r.email_address for r in message.cc_recipients]

        # Get timestamps
        received_time = message.datetime_received.isoformat() if message.datetime_received else ""
        sent_time = message.datetime_sent.isoformat() if message.datetime_sent else ""

        # Build HTML content
        html_content = self._build_email_html(
            subject=subject,
            sender_name=sender_name,
            sender_email=sender_email,
            to_recipients=to_recipients,
            cc_recipients=cc_recipients,
            sent_time=sent_time,
            received_time=received_time,
            body_content=body_content,
            body_type=body_type
        )

        # Create fully qualified source identifier
        qualified_source = f"exchange://{self.email_address}/{message_id}"

        # Construct metadata
        metadata = {
            "message_id": message_id,
            "internet_message_id": getattr(message, 'message_id', ''),
            "conversation_id": getattr(message, 'conversation_id', ''),
            "subject": subject,
            "sender_email": sender_email,
            "sender_name": sender_name,
            "to_recipients": to_recipients,
            "cc_recipients": cc_recipients,
            "received_datetime": received_time,
            "sent_datetime": sent_time,
            "has_attachments": getattr(message, 'has_attachments', False),
            "importance": getattr(message, 'importance', 'normal'),
            "content_type": "html",
            "api_type": "ews"
        }

        # Generate content hash for change detection
        content_hash = self.get_content_hash(html_content)

        # Cache the content
        self.content_cache[message_id] = {
            "content": html_content,
            "metadata": metadata,
            "hash": content_hash,
            "last_accessed": time.time()
        }

        return {
            "id": qualified_source,
            "content": html_content,
            "doc_type": "html",
            "metadata": metadata,
            "content_hash": content_hash
        }

    def _build_email_html(self, subject: str, sender_name: str, sender_email: str,
                          to_recipients: List[str], cc_recipients: List[str],
                          sent_time: str, received_time: str,
                          body_content: str, body_type: str) -> str:
        """Build HTML representation of email."""
        html_content = f"<h1>Email: {subject}</h1>\n"

        # Add email headers
        if self.include_headers:
            html_content += "<div class='email-headers'>\n"
            html_content += f"<p><strong>From:</strong> {sender_name} &lt;{sender_email}&gt;</p>\n"

            if to_recipients:
                to_list = ", ".join(to_recipients)
                html_content += f"<p><strong>To:</strong> {to_list}</p>\n"

            if cc_recipients:
                cc_list = ", ".join(cc_recipients)
                html_content += f"<p><strong>CC:</strong> {cc_list}</p>\n"

            if sent_time:
                formatted_sent = self._format_timestamp(sent_time)
                html_content += f"<p><strong>Sent:</strong> {formatted_sent}</p>\n"

            if received_time and received_time != sent_time:
                formatted_received = self._format_timestamp(received_time)
                html_content += f"<p><strong>Received:</strong> {formatted_received}</p>\n"

            html_content += "</div>\n"

        # Add email body
        if self.include_body and body_content:
            html_content += "<div class='email-body'>\n"

            if body_type == "html":
                # Clean up the HTML content
                cleaned_body = self._clean_html_content(body_content)
                html_content += cleaned_body
            else:
                # Convert plain text to HTML
                escaped_body = body_content.replace('&', '&amp;').replace('<', '&lt;').replace('>', '&gt;')
                html_content += f"<pre>{escaped_body}</pre>\n"

            html_content += "</div>\n"

        return html_content

    def _clean_html_content(self, html_content: str) -> str:
        """Clean and sanitize HTML content from email body."""
        if not BS4_AVAILABLE:
            return html_content

        try:
            soup = BeautifulSoup(html_content, 'html.parser')

            # Remove potentially dangerous elements
            for tag in soup.find_all(['script', 'style', 'link', 'meta']):
                tag.decompose()

            # Remove external references in images and links
            for img in soup.find_all('img'):
                if img.get('src') and img['src'].startswith(('http://', 'https://')):
                    img['src'] = '#'  # Replace with placeholder

            for link in soup.find_all('a'):
                if link.get('href') and not link['href'].startswith(('mailto:', '#')):
                    # Keep the link text but remove external href
                    link.replace_with(link.get_text())

            return str(soup)

        except Exception as e:
            logger.warning(f"Error cleaning HTML content: {str(e)}")
            return html_content

    def list_documents(self) -> List[Dict[str, Any]]:
        """
        List available emails in Exchange.

        Returns:
            List of email identifiers and metadata

        Raises:
            ValueError: If Exchange is not configured
        """
        logger.debug("Listing Exchange emails")

        try:
            if self.api_type == "graph" or self.client_id:
                return self._list_documents_graph()
            else:
                return self._list_documents_ews()

        except Exception as e:
            logger.error(f"Error listing Exchange emails: {str(e)}")
            raise

    def _list_documents_graph(self) -> List[Dict[str, Any]]:
        """List emails using Microsoft Graph API."""
        if not self.session:
            raise ValueError("Graph API not configured")

        self._ensure_valid_token()
        results = []

        try:
            # Calculate date filter
            if self.days_back > 0:
                from datetime import datetime, timedelta, timezone
                cutoff_date = datetime.now(timezone.utc) - timedelta(days=self.days_back)
                date_filter = cutoff_date.isoformat()
            else:
                date_filter = None

            # Process each configured folder
            for folder_name in self.folders:
                folder_results = self._list_folder_graph(folder_name, date_filter)
                results.extend(folder_results)

            logger.info(f"Found {len(results)} Exchange emails")
            return results[:self.max_results]

        except Exception as e:
            logger.error(f"Error listing emails via Graph API: {str(e)}")
            raise

    def _list_folder_graph(self, folder_name: str, date_filter: Optional[str]) -> List[Dict[str, Any]]:
        """List emails in a specific folder using Graph API."""
        results = []

        # Construct API URL
        mailbox_part = f"users/{self.mailbox or self.email_address}"

        if folder_name.lower() == "inbox":
            api_url = f"https://graph.microsoft.com/v1.0/{mailbox_part}/messages"
        else:
            api_url = f"https://graph.microsoft.com/v1.0/{mailbox_part}/mailFolders/{folder_name}/messages"

        # Set up headers
        headers = {
            "Authorization": f"Bearer {self.access_token}",
            "Content-Type": "application/json"
        }

        # Build filter query
        filter_parts = []
        if date_filter:
            filter_parts.append(f"receivedDateTime ge {date_filter}")
        if self.search_query:
            filter_parts.append(
                f"contains(subject,'{self.search_query}') or contains(body/content,'{self.search_query}')")

        # Set up parameters
        params = {
            "$select": "id,subject,sender,from,receivedDateTime,sentDateTime,hasAttachments,internetMessageId,conversationId",
            "$orderby": "receivedDateTime desc",
            "$top": min(100, self.max_results)
        }

        if filter_parts:
            params["$filter"] = " and ".join(filter_parts)

        # Make paginated requests
        while api_url and len(results) < self.max_results:
            response = self.session.get(api_url, headers=headers, params=params)
            response.raise_for_status()
            data = response.json()

            messages = data.get("value", [])
            if not messages:
                break

            for message in messages:
                if len(results) >= self.max_results:
                    break

                message_id = message.get("id", "")
                subject = message.get("subject", "")

                # Get sender information
                sender_data = message.get("sender", {}) or message.get("from", {})
                sender_email = sender_data.get("emailAddress", {}).get("address", "")
                sender_name = sender_data.get("emailAddress", {}).get("name", "")

                # Create fully qualified source identifier
                qualified_source = f"exchange://{self.email_address}/{message_id}"

                # Create metadata
                metadata = {
                    "message_id": message_id,
                    "internet_message_id": message.get("internetMessageId", ""),
                    "conversation_id": message.get("conversationId", ""),
                    "subject": subject,
                    "sender_email": sender_email,
                    "sender_name": sender_name,
                    "received_datetime": message.get("receivedDateTime", ""),
                    "sent_datetime": message.get("sentDateTime", ""),
                    "has_attachments": message.get("hasAttachments", False),
                    "folder": folder_name,
                    "content_type": "html",
                    "api_type": "graph"
                }

                results.append({
                    "id": qualified_source,
                    "metadata": metadata,
                    "doc_type": "html"
                })

            # Get next page URL
            api_url = data.get("@odata.nextLink")
            params = {}  # Parameters are included in the nextLink URL

        return results

    def _list_documents_ews(self) -> List[Dict[str, Any]]:
        """List emails using Exchange Web Services."""
        if not self.ews_account:
            raise ValueError("EWS not configured")

        results = []

        try:
            # Calculate date filter
            if self.days_back > 0:
                from exchangelib import EWSDateTime
                from datetime import datetime, timedelta, timezone
                cutoff_date = datetime.now(timezone.utc) - timedelta(days=self.days_back)
                ews_cutoff = EWSDateTime.from_datetime(cutoff_date)
            else:
                ews_cutoff = None

            # Process each configured folder
            for folder_name in self.folders:
                folder_results = self._list_folder_ews(folder_name, ews_cutoff)
                results.extend(folder_results)

            logger.info(f"Found {len(results)} Exchange emails")
            return results[:self.max_results]

        except Exception as e:
            logger.error(f"Error listing emails via EWS: {str(e)}")
            raise

    def _list_folder_ews(self, folder_name: str, date_filter) -> List[Dict[str, Any]]:
        """List emails in a specific folder using EWS."""
        results = []

        try:
            # Get the folder
            if folder_name.lower() == "inbox":
                folder = self.ews_account.inbox
            else:
                # Try to find folder by name
                folder = None
                for f in self.ews_account.root.walk():
                    if f.name.lower() == folder_name.lower():
                        folder = f
                        break

                if not folder:
                    logger.warning(f"Folder not found: {folder_name}")
                    return results

            # Build query
            query = folder.all()

            if date_filter:
                query = query.filter(datetime_received__gte=date_filter)

            if self.search_query:
                query = query.filter(subject__contains=self.search_query)

            # Order by received date and limit results
            query = query.order_by('-datetime_received')[:self.max_results]

            # Fetch and process messages
            for message in query:
                message_id = message.id
                subject = message.subject or ""

                # Get sender information
                sender_email = ""
                sender_name = ""
                if hasattr(message, 'sender') and message.sender:
                    sender_email = message.sender.email_address
                    sender_name = message.sender.name or sender_email

                # Create fully qualified source identifier
                qualified_source = f"exchange://{self.email_address}/{message_id}"

                # Create metadata
                metadata = {
                    "message_id": message_id,
                    "internet_message_id": getattr(message, 'message_id', ''),
                    "conversation_id": getattr(message, 'conversation_id', ''),
                    "subject": subject,
                    "sender_email": sender_email,
                    "sender_name": sender_name,
                    "received_datetime": message.datetime_received.isoformat() if message.datetime_received else "",
                    "sent_datetime": message.datetime_sent.isoformat() if message.datetime_sent else "",
                    "has_attachments": getattr(message, 'has_attachments', False),
                    "folder": folder_name,
                    "content_type": "html",
                    "api_type": "ews"
                }

                results.append({
                    "id": qualified_source,
                    "metadata": metadata,
                    "doc_type": "html"
                })

                if len(results) >= self.max_results:
                    break

        except Exception as e:
            logger.error(f"Error listing folder {folder_name} via EWS: {str(e)}")

        return results

    def has_changed(self, source_id: str, last_modified: Optional[float] = None) -> bool:
        """
        Check if an email has changed since last processing.

        Args:
            source_id: Identifier for the email
            last_modified: Timestamp of last known modification

        Returns:
            True if email has changed, False otherwise
        """
        logger.debug(f"Checking if Exchange email has changed: {source_id}")

        # Emails generally don't change after they're received, but we can check
        # the received timestamp to see if it's newer than our last check
        try:
            message_id = self._extract_message_id(source_id)

            # Check cache first
            if message_id in self.content_cache:
                cache_entry = self.content_cache[message_id]
                cache_metadata = cache_entry.get("metadata", {})

                received_time = cache_metadata.get("received_datetime", "")
                if received_time and last_modified:
                    received_timestamp = self._parse_timestamp(received_time)
                    if received_timestamp and received_timestamp <= last_modified:
                        logger.debug(f"Email {message_id} unchanged according to cache")
                        return False

            # For emails, we generally consider them unchanged once processed
            # unless there's a specific reason to believe they've changed
            return last_modified is None

        except Exception as e:
            logger.error(f"Error checking changes for {source_id}: {str(e)}")
            return True  # Assume changed if there's an error

    def follow_links(self, content: str, source_id: str, current_depth: int = 0,
                     global_visited_docs=None) -> List[Dict[str, Any]]:
        """
        Extract and follow links to related emails (conversation threads).

        Args:
            content: Email content (HTML format)
            source_id: Identifier for the source email
            current_depth: Current depth of link following
            global_visited_docs: Global set of all visited email IDs

        Returns:
            List of related emails

        Raises:
            ValueError: If Exchange is not configured
        """
        if current_depth >= self.max_link_depth:
            logger.debug(f"Max link depth {self.max_link_depth} reached for {source_id}")
            return []

        # Initialize global visited set if not provided
        if global_visited_docs is None:
            global_visited_docs = set()

        # Add current email to global visited set
        global_visited_docs.add(source_id)

        logger.debug(f"Following links in Exchange email {source_id} at depth {current_depth}")

        linked_docs = []
        message_id = self._extract_message_id(source_id)

        try:
            # Get conversation thread if enabled
            if self.include_thread:
                thread_emails = self._get_conversation_thread(message_id)

                for thread_email_id in thread_emails:
                    qualified_id = f"exchange://{self.email_address}/{thread_email_id}"

                    if qualified_id in global_visited_docs or thread_email_id in global_visited_docs:
                        logger.debug(f"Skipping globally visited email: {thread_email_id}")
                        continue

                    global_visited_docs.add(qualified_id)
                    global_visited_docs.add(thread_email_id)

                    try:
                        # Fetch the related email
                        linked_doc = self.fetch_document(thread_email_id)
                        linked_docs.append(linked_doc)
                        logger.debug(f"Successfully fetched thread email: {thread_email_id}")

                        # Recursively follow links if not at max depth
                        if current_depth + 1 < self.max_link_depth:
                            nested_docs = self.follow_links(
                                linked_doc["content"],
                                linked_doc["id"],
                                current_depth + 1,
                                global_visited_docs
                            )
                            linked_docs.extend(nested_docs)

                    except Exception as e:
                        logger.warning(f"Error following link {thread_email_id} from {source_id}: {str(e)}")

            logger.debug(f"Completed following links from {source_id}: found {len(linked_docs)} related emails")
            return linked_docs

        except Exception as e:
            logger.error(f"Error following links from Exchange email {source_id}: {str(e)}")
            return []

    def _get_conversation_thread(self, message_id: str) -> List[str]:
        """Get emails in the same conversation thread."""
        thread_emails = []

        try:
            if self.api_type == "graph" or self.client_id:
                thread_emails = self._get_conversation_thread_graph(message_id)
            else:
                thread_emails = self._get_conversation_thread_ews(message_id)

        except Exception as e:
            logger.warning(f"Error getting conversation thread for {message_id}: {str(e)}")

        return thread_emails

    def _get_conversation_thread_graph(self, message_id: str) -> List[str]:
        """Get conversation thread using Graph API."""
        if not self.session:
            return []

        self._ensure_valid_token()
        thread_emails = []

        try:
            # First get the conversation ID of the current message
            mailbox_part = f"users/{self.mailbox or self.email_address}"
            api_url = f"https://graph.microsoft.com/v1.0/{mailbox_part}/messages/{message_id}"

            headers = {
                "Authorization": f"Bearer {self.access_token}",
                "Content-Type": "application/json"
            }

            params = {"$select": "conversationId"}

            response = self.session.get(api_url, headers=headers, params=params)
            response.raise_for_status()
            message_data = response.json()

            conversation_id = message_data.get("conversationId")
            if not conversation_id:
                return []

            # Get all messages in this conversation
            api_url = f"https://graph.microsoft.com/v1.0/{mailbox_part}/messages"
            params = {
                "$filter": f"conversationId eq '{conversation_id}'",
                "$select": "id",
                "$orderby": "receivedDateTime asc"
            }

            response = self.session.get(api_url, headers=headers, params=params)
            response.raise_for_status()
            data = response.json()

            for message in data.get("value", []):
                email_id = message.get("id")
                if email_id and email_id != message_id:  # Don't include the original message
                    thread_emails.append(email_id)

        except Exception as e:
            logger.warning(f"Error getting Graph conversation thread: {str(e)}")

        return thread_emails

    def _get_conversation_thread_ews(self, message_id: str) -> List[str]:
        """Get conversation thread using EWS."""
        if not self.ews_account:
            return []

        thread_emails = []

        try:
            # Get the original message to find its conversation ID
            from exchangelib import Message
            message = Message.get(item_id=message_id, account=self.ews_account)

            conversation_id = getattr(message, 'conversation_id', None)
            if not conversation_id:
                return []

            # Find all messages with the same conversation ID
            messages = self.ews_account.inbox.filter(
                conversation_id=conversation_id
            ).order_by('datetime_received')

            for msg in messages:
                if msg.id != message_id:  # Don't include the original message
                    thread_emails.append(msg.id)

        except Exception as e:
            logger.warning(f"Error getting EWS conversation thread: {str(e)}")

        return thread_emails

    @staticmethod
    def _extract_message_id(source_id: str) -> str:
        """
        Extract message ID from source ID.

        Args:
            source_id: Source identifier

        Returns:
            Message ID
        """
        # If source_id looks like a message ID already, return it directly
        if not source_id.startswith("exchange://"):
            return source_id

        # Extract from fully qualified source identifier
        # Pattern: exchange://email@domain.com/message_id
        match = re.search(r'exchange://[^/]+/(.+)', source_id)
        if match:
            return match.group(1)

        return source_id

    @staticmethod
    def _parse_timestamp(timestamp: str) -> Optional[float]:
        """
        Parse timestamp into epoch time.

        Args:
            timestamp: Timestamp string

        Returns:
            Timestamp as epoch time or None if parsing fails
        """
        if not timestamp:
            return None

        try:
            if DATEUTIL_AVAILABLE:
                dt = dateutil.parser.parse(timestamp)
                return dt.timestamp()
            else:
                # Fallback for when dateutil is not available
                from datetime import datetime
                # Try common ISO format
                try:
                    dt = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
                    return dt.timestamp()
                except ValueError:
                    return None
        except Exception:
            logger.warning(f"Could not parse timestamp: {timestamp}")
            return None

    @staticmethod
    def _format_timestamp(timestamp: str) -> str:
        """
        Format timestamp into readable format.

        Args:
            timestamp: Timestamp string

        Returns:
            Formatted timestamp
        """
        if not timestamp:
            return ""

        try:
            if DATEUTIL_AVAILABLE:
                dt = dateutil.parser.parse(timestamp)
                return dt.strftime("%Y-%m-%d %H:%M")
            else:
                from datetime import datetime
                dt = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
                return dt.strftime("%Y-%m-%d %H:%M")
        except Exception:
            return timestamp

    def __del__(self):
        """Close connections when object is deleted."""
        if self.session:
            try:
                self.session.close()
                logger.debug("Closed Exchange session")
            except Exception as e:
                logger.warning(f"Error closing Exchange session: {str(e)}")
