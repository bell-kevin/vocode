import { Outlet } from "react-router-dom";

function Root() {
    return <> 
        <header className="bg-black text-white p-[1.5rem] pb-[2rem] text-left">
            <h1>Vocode</h1>
        </header>
        <main className="min-h-[calc(100vh - 70px)]">
            <Outlet/>
        </main>
        <footer className="bg-black text-white p-[2rem] pb-[3rem] h-50px">
            © 2026 Vocode. All Rights Reserved.
        </footer>
    </>
}

export default Root;