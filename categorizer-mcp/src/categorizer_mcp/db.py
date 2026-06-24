"""Postgres access for the categorizer.

These queries mirror the Go ``repository`` package against the same schema:
``products``, ``categories``, ``product_categories`` and the lower(name) reuse
cache. A small connection pool keeps per-request latency down.
"""

from __future__ import annotations

from dataclasses import dataclass

from psycopg_pool import ConnectionPool

from . import ocr


@dataclass(frozen=True)
class Category:
    id: int
    name: str


@dataclass(frozen=True)
class Product:
    id: int
    name: str
    price: int
    receipt_id: int


@dataclass(frozen=True)
class SimilarProduct:
    # Original (accented) product name — never normalized — plus its known category
    # and the OCR-aware match score, so the model reasons over the real spelling.
    name: str
    category: str
    score: float


class Database:
    def __init__(self, dsn: str) -> None:
        # min_size=0 + open=False so startup never blocks and no connection is
        # made until a DB-backed tool actually runs; the pure categorize tool
        # then works even when Postgres is down.
        self._pool = ConnectionPool(conninfo=dsn, min_size=0, max_size=4, open=False)

    def open(self) -> None:
        self._pool.open()

    def close(self) -> None:
        self._pool.close()

    def ping(self, timeout: float = 3.0) -> None:
        # Short timeout so a down DB doesn't stall startup; the pure categorize
        # tool serves regardless and DB tools reconnect on demand.
        with self._pool.connection(timeout=timeout) as conn, conn.cursor() as cur:
            cur.execute("SELECT 1")
            cur.fetchone()

    def all_categories(self) -> list[Category]:
        """The fixed category set the model is constrained to choose from."""
        with self._pool.connection() as conn, conn.cursor() as cur:
            cur.execute("SELECT id, name FROM categories ORDER BY id")
            return [Category(id=r[0], name=r[1]) for r in cur.fetchall()]

    def uncategorized_products(self, receipt_id: int | None = None) -> list[Product]:
        """Products with no row in product_categories yet, optionally one receipt."""
        sql = """
            SELECT p.id, p.name, p.price, p.receipt_id
            FROM products p
            LEFT JOIN product_categories pc ON pc.product_id = p.id
            WHERE pc.product_id IS NULL
        """
        params: tuple = ()
        if receipt_id is not None:
            sql += " AND p.receipt_id = %s"
            params = (receipt_id,)
        sql += " ORDER BY p.id"
        with self._pool.connection() as conn, conn.cursor() as cur:
            cur.execute(sql, params)
            return [
                Product(id=r[0], name=r[1], price=r[2], receipt_id=r[3])
                for r in cur.fetchall()
            ]

    def find_category_for_name(self, name: str) -> tuple[int, str] | None:
        """Reuse cache: the category of the most recently categorized product of
        the same name (case-insensitive), so repeat purchases skip the LLM."""
        with self._pool.connection() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT c.id, c.name
                FROM products p
                JOIN product_categories pc ON pc.product_id = p.id
                JOIN categories c ON c.id = pc.category_id
                WHERE lower(p.name) = lower(%s)
                ORDER BY p.id DESC
                LIMIT 1
                """,
                (name,),
            )
            row = cur.fetchone()
            return (row[0], row[1]) if row else None

    def search_similar_products(
        self, name: str, limit: int = 5, min_score: float = 60.0
    ) -> list[SimilarProduct]:
        """Find already-categorized products whose names look like ``name`` despite
        OCR noise — the agent's main tool for adding context and catching
        hallucinations on a mangled name.

        Ranking is done in Python over the OCR-folded key (see ``ocr.similarity``)
        rather than in SQL, so digit/letter swaps and Hungarian accents don't sink a
        match. The names returned are the original accented spellings, never folded.
        On a personal-finance dataset the categorized set is small enough to score in
        memory; switch to pg_trgm if it ever grows large.
        """
        with self._pool.connection() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT DISTINCT ON (lower(p.name)) p.name, c.name
                FROM products p
                JOIN product_categories pc ON pc.product_id = p.id
                JOIN categories c ON c.id = pc.category_id
                ORDER BY lower(p.name), p.id DESC
                """
            )
            rows = cur.fetchall()

        scored = (
            SimilarProduct(name=r[0], category=r[1], score=ocr.similarity(name, r[0]))
            for r in rows
        )
        ranked = sorted(
            (s for s in scored if s.score >= min_score),
            key=lambda s: s.score,
            reverse=True,
        )
        return ranked[:limit]

    def assign_categories(self, pairs: list[tuple[int, int]]) -> int:
        """Commit (product_id, category_id) assignments in one transaction,
        idempotently. Returns the number of newly inserted rows."""
        if not pairs:
            return 0
        inserted = 0
        with self._pool.connection() as conn:
            with conn.cursor() as cur:
                for product_id, category_id in pairs:
                    cur.execute(
                        """
                        INSERT INTO product_categories (product_id, category_id)
                        VALUES (%s, %s)
                        ON CONFLICT (product_id, category_id) DO NOTHING
                        """,
                        (product_id, category_id),
                    )
                    inserted += cur.rowcount
            conn.commit()
        return inserted
