-- Structured content shown with a Local Guide reply: the trip-planner's
-- question card, a visual trip plan, or an "add to calendar" action. The
-- app renders it as cards; replayed to the model as a short text summary.
ALTER TABLE guide_messages ADD COLUMN payload JSONB;
