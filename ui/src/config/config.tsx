import axios from "axios";

class StoreDatas{
    // Dev (`npm run dev`) targets the local API; production build (`npm run build`) uses same-origin.
    server_ip: string = import.meta.env.DEV ? "http://localhost:8001" : "";

    accessToken: string = ""
    auth: boolean = false;
    id: number = 0;
    login: string = "";
    name: string = "";
    role: string = "";
    role_id: number | null = null;
    agentAllowed: boolean = false;

    setAccessToken(accessToken: string) {
        this.accessToken = accessToken;
    }

    getAccessToken() {
        return this.accessToken;
    }

    isAuth(){
        return this.auth;
    }

    getLogin(){
        return this.login;
    }

    isAdmin(){
        return this.role.toLowerCase() === "admin";
    }

    setAgentAllowed(allowed: boolean) {
        this.agentAllowed = allowed;
        localStorage.setItem("agentAllowed", String(allowed));
    }

    isAgentOwner() {
        return this.agentAllowed;
    }

    visibleNavSections<T extends { adminOnly?: boolean; agentOwnerOnly?: boolean }>(sections: T[]): T[] {
        return sections.filter((section) => {
            if (section.adminOnly && !this.isAdmin()) return false;
            if (section.agentOwnerOnly && !this.isAgentOwner()) return false;
            return true;
        });
    }

    loadLocalData(){
        this.auth = Boolean(localStorage.getItem("isauth"));
        this.accessToken = localStorage.getItem("accessToken") || "";
        axios.defaults.headers.common.Authorization = `${this.accessToken}`
        this.role = localStorage.getItem("role") || "";
        this.id = Number(localStorage.getItem("id")) || 0;
        this.login = localStorage.getItem("login") || "";
        this.name = localStorage.getItem("name") || "";
        this.role_id = Number(localStorage.getItem("role_id")) || null;
        this.agentAllowed = localStorage.getItem("agentAllowed") === "true";
    }

    setUserData(isAuth: boolean, accessToken: string, role: string, id: number, 
        login: string, name: string, role_id: number | null){
        this.auth = isAuth;
        this.accessToken = accessToken;
        this.role = role;
        this.id = id;
        this.login = login;
        this.name = name;
        this.role_id = role_id;
        localStorage.setItem("isauth", String(true));
        localStorage.setItem("accessToken", accessToken);
        localStorage.setItem("role", role);
        localStorage.setItem("id", String(id));
        localStorage.setItem("login", login);
        localStorage.setItem("name", name);
        localStorage.setItem("role_id", String(role_id));
    }

    clearUserData(){
        this.auth = false;
        this.accessToken = "";
        this.role = "user";
        this.id = 0;
        this.login = "";
        this.name = "";
        this.role_id = null;
        this.agentAllowed = false;
        localStorage.removeItem("isauth");
        localStorage.removeItem("accessToken");
        localStorage.removeItem("role");
        localStorage.removeItem("id");
        localStorage.removeItem("login");
        localStorage.removeItem("name");
        localStorage.removeItem("role_id");
        localStorage.removeItem("agentAllowed");
    }
} 

export const Global_Data = new StoreDatas();