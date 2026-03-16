"""
UbiGo Python Client
-------------------
A comprehensive, type-annotated, and robust Python client for the Ubi-Go API backend.
Handles authentication, stats, profiles, and error management.
"""

import requests
from typing import Any, Dict, List, Optional, Union

class UbiGoClientError(Exception):
    """Base exception for UbiGoClient errors."""
    pass

class UbiGoAPIError(UbiGoClientError):
    """Raised when the API returns an error response."""
    def __init__(self, status_code: int, message: str, response: Optional[requests.Response] = None):
        self.status_code = status_code
        self.message = message
        self.response = response
        super().__init__(f"API Error {status_code}: {message}")

class UbiGoClient:
    """
    Python client for the Ubi-Go API backend.
    """
    def __init__(self, base_url: str, timeout: int = 10):
        """
        Args:
            base_url: The root URL of the Ubi-Go API (e.g., http://localhost:8080)
            timeout: Timeout for HTTP requests in seconds.
        """
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.session = requests.Session()

    def _handle_response(self, resp: requests.Response) -> Any:
        try:
            data = resp.json()
        except Exception as e:
            raise UbiGoAPIError(resp.status_code, f"Invalid JSON response: {e}", resp)
        if not data.get("success", False):
            raise UbiGoAPIError(resp.status_code, data.get("error", "Unknown error"), resp)
        return data.get("data")

    def health(self) -> bool:
        """Check if the API is healthy."""
        url = f"{self.base_url}/health"
        resp = self.session.get(url, timeout=self.timeout)
        return resp.status_code == 200 and resp.json().get("success", False)

    def get_profile(self, uid: Optional[str] = None, username: Optional[str] = None, platform: Optional[str] = None) -> Dict[str, Any]:
        """
        Retrieve a user profile by UID or username+platform.
        Raises UbiGoAPIError on error.
        """
        url = f"{self.base_url}/profile"
        params = {}
        if uid:
            params["uid"] = uid
        if username:
            params["username"] = username
        if platform:
            params["platform"] = platform
        resp = self.session.get(url, params=params, timeout=self.timeout)
        return self._handle_response(resp)

    def get_stats(self, game_id: str, platform: str, uids: Union[str, List[str]]) -> List[Dict[str, Any]]:
        """
        Retrieve stats for one or more player IDs.
        Args:
            game_id: The game/space ID.
            platform: Platform type (e.g., 'uplay', 'xbl', 'psn').
            uids: A single UID or a list of UIDs.
        Returns:
            List of stats dicts.
        Raises UbiGoAPIError on error.
        """
        if isinstance(uids, list):
            uids_param = ",".join(uids)
        else:
            uids_param = uids
        url = f"{self.base_url}/stats"
        params = {"gameId": game_id, "platform": platform, "uids": uids_param}
        resp = self.session.get(url, params=params, timeout=self.timeout)
        return self._handle_response(resp)

    def get_statscard(self, uid: str, space_id: str) -> Dict[str, Any]:
        """
        Retrieve statscard for a given UID and spaceId.
        Raises UbiGoAPIError on error.
        """
        url = f"{self.base_url}/statscard"
        params = {"uid": uid, "spaceId": space_id}
        resp = self.session.get(url, params=params, timeout=self.timeout)
        return self._handle_response(resp)

    def check_username(self, username: str, platform: str) -> bool:
        """
        Check if a username is available (if implemented on backend).
        Returns True if available, False if taken or not implemented.
        """
        url = f"{self.base_url}/username/check"
        params = {"username": username, "platform": platform}
        resp = self.session.get(url, params=params, timeout=self.timeout)
        if resp.status_code == 501:
            return False
        try:
            data = resp.json()
            return data.get("success", False) and data.get("data", False)
        except Exception:
            return False

    def get_games(self) -> List[str]:
        """
        Retrieve the list of supported games.
        Raises UbiGoAPIError on error.
        """
        url = f"{self.base_url}/games"
        resp = self.session.get(url, timeout=self.timeout)
        return self._handle_response(resp)

    def root_api_info(self) -> Dict[str, Any]:
        """
        Get the root API index (endpoints and docs).
        """
        url = f"{self.base_url}/"
        resp = self.session.get(url, timeout=self.timeout)
        try:
            return resp.json()
        except Exception as e:
            raise UbiGoAPIError(resp.status_code, f"Invalid JSON response: {e}", resp)

    def report(self, payload: dict) -> dict:
            """
            Send a Ubisoft player report using a POST request to the /report endpoint.
            Args:
                payload: The report data to send (must be a dict).
            Returns:
                dict: API response or error details.
            Raises:
                ValueError: If payload is not provided.
                UbiGoAPIError: If the API returns an error response.
            """
            if not payload:
                raise ValueError("Payload was never provided")

            url = f"{self.base_url}/report"
            headers = {
                "Content-Type": "application/json; charset=utf-8",
                "Accept": "application/json, text/plain, */*",
            }
            try:
                resp = self.session.post(url, json=payload, headers=headers, timeout=self.timeout)
            except Exception as e:
                raise UbiGoClientError(f"Failed to send report: {e}")

            try:
                data = resp.json()
            except Exception as e:
                raise UbiGoAPIError(resp.status_code, f"Invalid JSON response: {e}", resp)

            if not data.get("success", False):
                return {"status": False, "error": data.get("error", "Unknown error"), "code": resp.status_code}

            if "errorCode" in data:
                return {"error": data.get("message", "An unknown error occurred.")}

            return {"data": data.get("data")}
        
    def close(self) -> None:
        """Close the underlying HTTP session."""
        self.session.close()
